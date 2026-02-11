import os
import sqlite3
from pathlib import Path

from dagster import (
    AssetExecutionContext,
    MaterializeResult,
    RunRequest,
    SkipReason,
    asset,
    sensor,
)


def _latest_snapshot_path() -> Path:
    snapshot_dir = os.environ.get("HAA_SNAPSHOT_DIR")
    if not snapshot_dir:
        snapshot_dir = Path(__file__).resolve().parents[2] / "aggregation" / "snapshot"
        snapshot_dir = str(snapshot_dir)
    if not snapshot_dir:
        raise RuntimeError("HAA_SNAPSHOT_DIR is not set")

    root = Path(snapshot_dir)
    if not root.exists() or not root.is_dir():
        raise RuntimeError(f"HAA_SNAPSHOT_DIR is not a directory: {snapshot_dir}")

    candidates = list(root.glob("*.sqlite"))
    if not candidates:
        raise RuntimeError(f"no snapshot files found in {snapshot_dir}")

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
def snapshot_sensor():
    try:
        snapshot_path = _latest_snapshot_path()
    except RuntimeError as exc:
        return SkipReason(str(exc))

    run_key = str(snapshot_path)
    return RunRequest(run_key=run_key, run_config={})
