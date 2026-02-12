import sqlite3
import json
import os
import urllib.error
import urllib.request
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
    return _latest_snapshot_path_in_dir(_snapshot_root())


def _latest_testing_snapshot_path() -> Path:
    return _latest_snapshot_path_in_dir(_testing_snapshot_root())


def _snapshot_root() -> Path:
    return _repository_root() / "data" / "snapshots"


def _testing_snapshot_root() -> Path:
    return _repository_root() / "test-data" / "snapshots"


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


def _testing_api_base_url() -> str:
    return os.getenv("HAA_TESTING_API_URL", "http://127.0.0.1:8081").rstrip("/")


def _testing_flow_test_name() -> str:
    return os.getenv("HAA_DAGSTER_TEST_NAME", "dagster-asset-flow")


def _testing_flow_snapshot_name() -> str:
    return os.getenv("HAA_DAGSTER_SNAPSHOT_NAME", "dagster-asset-flow")


def _post_json(url: str, payload: dict) -> tuple[int, dict]:
    req = urllib.request.Request(
        url=url,
        data=json.dumps(payload).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=10) as resp:
        body = resp.read().decode("utf-8")
        parsed = json.loads(body) if body else {}
        return resp.status, parsed


def _require_status(status: int, expected: int, step: str, payload: dict) -> None:
    if status != expected:
        raise RuntimeError(f"{step} failed: expected {expected}, got {status}, payload={payload}")


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
        try:
            cur.execute("SELECT COUNT(*) FROM controls")
            controls_count = cur.fetchone()[0]
            cur.execute("SELECT COUNT(*) FROM aggregates")
            aggregates_count = cur.fetchone()[0]
        except (sqlite3.OperationalError, sqlite3.DatabaseError) as exc:
            message = f"failed snapshot query for {snapshot_path}: {exc}"
            context.log.error(message)
            raise RuntimeError(message) from exc
    finally:
        conn.close()

    return MaterializeResult(
        metadata={
            "snapshot_path": str(snapshot_path),
            "controls_count": controls_count,
            "aggregates_count": aggregates_count,
        }
    )


@asset
def testing_snapshot_summary(context: AssetExecutionContext) -> MaterializeResult:
    try:
        snapshot_path = _latest_testing_snapshot_path()
    except RuntimeError as exc:
        context.log.warning(str(exc))
        return MaterializeResult(metadata={"snapshot_missing": True})

    context.log.info("using testing snapshot %s", snapshot_path)

    conn = sqlite3.connect(snapshot_path)
    try:
        cur = conn.cursor()
        try:
            cur.execute("SELECT COUNT(*) FROM controls")
            controls_count = cur.fetchone()[0]
            cur.execute("SELECT COUNT(*) FROM aggregates")
            aggregates_count = cur.fetchone()[0]
        except (sqlite3.OperationalError, sqlite3.DatabaseError) as exc:
            message = f"failed snapshot query for {snapshot_path}: {exc}"
            context.log.error(message)
            raise RuntimeError(message) from exc
    finally:
        conn.close()

    return MaterializeResult(
        metadata={
            "snapshot_path": str(snapshot_path),
            "controls_count": controls_count,
            "aggregates_count": aggregates_count,
        }
    )


@asset
def testing_api_snapshot_validation(context: AssetExecutionContext) -> MaterializeResult:
    base_url = _testing_api_base_url()
    test_name = _testing_flow_test_name()
    snapshot_name = _testing_flow_snapshot_name()

    try:
        status, payload = _post_json(f"{base_url}/v1/reset", {"testName": test_name})
        _require_status(status, 200, "reset", payload)

        status, payload = _post_json(
            f"{base_url}/v1/controls",
            {"testName": test_name, "controlId": "c1", "controlType": "discrete", "numStates": 2},
        )
        _require_status(status, 202, "create control c1", payload)
        status, payload = _post_json(
            f"{base_url}/v1/controls",
            {"testName": test_name, "controlId": "c2", "controlType": "discrete", "numStates": 2},
        )
        _require_status(status, 202, "create control c2", payload)

        status, payload = _post_json(
            f"{base_url}/v1/holding-intervals",
            {
                "testName": test_name,
                "controlId": "c1",
                "modelId": "m1",
                "state": 1,
                "startTimeMs": 1578268800000,
                "endTimeMs": 1578269100000,
            },
        )
        _require_status(status, 202, "holding c1", payload)
        status, payload = _post_json(
            f"{base_url}/v1/holding-intervals",
            {
                "testName": test_name,
                "controlId": "c2",
                "modelId": "m1",
                "state": 0,
                "startTimeMs": 1578269100000,
                "endTimeMs": 1578269400000,
            },
        )
        _require_status(status, 202, "holding c2", payload)

        status, payload = _post_json(
            f"{base_url}/v1/snapshots",
            {"testName": test_name, "snapshotName": snapshot_name},
        )
        _require_status(status, 200, "snapshot export", payload)
    except urllib.error.URLError as exc:
        message = f"testing API unavailable at {base_url}: {exc}"
        context.log.warning(message)
        return MaterializeResult(metadata={"snapshot_missing": True, "error": message})

    snapshot_path = _testing_snapshot_root() / f"{test_name}-{snapshot_name}-snapshot.sqlite"
    if not snapshot_path.exists():
        raise RuntimeError(f"expected snapshot not found after export: {snapshot_path}")

    conn = sqlite3.connect(snapshot_path)
    try:
        controls_count = conn.execute("SELECT COUNT(*) FROM controls").fetchone()[0]
        aggregates_count = conn.execute("SELECT COUNT(*) FROM aggregates").fetchone()[0]
    finally:
        conn.close()

    if controls_count < 2:
        raise RuntimeError(f"snapshot should contain at least 2 controls, got {controls_count}")
    if aggregates_count < 2:
        raise RuntimeError(f"snapshot should contain at least 2 aggregates, got {aggregates_count}")

    return MaterializeResult(
        metadata={
            "testing_api_url": base_url,
            "snapshot_path": str(snapshot_path),
            "controls_count": controls_count,
            "aggregates_count": aggregates_count,
            "test_name": test_name,
            "snapshot_name": snapshot_name,
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
