import sqlite3
from pathlib import Path

from dagster import (
    AssetExecutionContext,
    MaterializeResult,
    RunRequest,
    SensorEvaluationContext,
    SkipReason,
    asset,
    sensor,
)


def _latest_snapshot_path() -> Path:
    root = _repository_root() / "aggregation" / "data" / "snapshots"
    return _latest_snapshot_path_in_dir(root)


def _repository_root() -> Path:
    start = Path(__file__).resolve()
    for parent in start.parents:
        if (parent / "aggregation").is_dir():
            return parent
    raise RuntimeError(f"repository root not found from {start}")


def _latest_snapshot_path_in_dir(root: Path) -> Path:
    if not root.exists() or not root.is_dir():
        raise RuntimeError(f"snapshot directory is not present: {root}")

    candidates = list(root.glob("*.sqlite"))
    if not candidates:
        raise RuntimeError(f"no snapshot files found in {root}")

    return max(candidates, key=lambda p: p.stat().st_mtime)


@asset
def snapshot_summary(context: AssetExecutionContext) -> MaterializeResult:
    try:
        snapshot_path = _latest_snapshot_path()
    except RuntimeError as exc:
        context.log.warning(str(exc))
        return MaterializeResult(metadata={"snapshot_missing": True})

    context.log.info("using snapshot %s", snapshot_path)

    conn = sqlite3.connect(snapshot_path)
    try:
        cur = conn.cursor()
        cur.execute("SELECT COUNT(*) FROM controls")
        controls_count = cur.fetchone()[0]
        cur.execute("SELECT COUNT(*) FROM aggregates")
        aggregates_count = cur.fetchone()[0]
    finally:
        conn.close()

    return MaterializeResult(
        metadata={
            "snapshot_path": str(snapshot_path),
            "controls_count": controls_count,
            "aggregates_count": aggregates_count,
        }
    )


@sensor(job_name="snapshot_job")
def snapshot_sensor(context: SensorEvaluationContext):
    try:
        snapshot_path = _latest_snapshot_path()
    except RuntimeError as exc:
        return SkipReason(str(exc))

    mtime_ns = snapshot_path.stat().st_mtime_ns
    cursor_raw = context.cursor or ""
    last_seen_mtime = -1
    if cursor_raw:
        try:
            last_seen_mtime = int(cursor_raw)
        except ValueError:
            last_seen_mtime = -1

    if mtime_ns <= last_seen_mtime:
        return SkipReason("no new snapshot detected")

    context.update_cursor(str(mtime_ns))
    run_key = f"{snapshot_path}:{mtime_ns}"
    return RunRequest(
        run_key=run_key,
        run_config={},
        tags={"snapshot_path": str(snapshot_path), "snapshot_mtime_ns": str(mtime_ns)},
    )
