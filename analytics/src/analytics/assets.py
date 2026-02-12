"""Dagster assets and sensors for snapshot validation and reporting."""

import sqlite3
import json
import os
import urllib.error
import urllib.request
from contextlib import closing
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
    """Return the newest production snapshot file path."""
    return _latest_snapshot_path_in_dir(_snapshot_root())


def _latest_testing_snapshot_path() -> Path:
    """Return the newest testing snapshot file path."""
    return _latest_snapshot_path_in_dir(_testing_snapshot_root())


def _snapshot_root() -> Path:
    """Return the production snapshot directory."""
    return _repository_root() / "data" / "snapshots"


def _testing_snapshot_root() -> Path:
    """Return the testing snapshot directory."""
    return _repository_root() / "test-data" / "snapshots"


def _repository_root() -> Path:
    """Resolve the monorepo root by searching parent directories."""
    start = Path(__file__).resolve()
    analytics_only_candidate: Path | None = None
    for parent in start.parents:
        has_aggregation = (parent / "aggregation").is_dir()
        has_analytics = (parent / "analytics").is_dir()
        if has_aggregation or has_analytics:
            if has_aggregation:
                return parent
            if analytics_only_candidate is None:
                analytics_only_candidate = parent
    if analytics_only_candidate is not None:
        return analytics_only_candidate
    raise RuntimeError(f"repository root not found from {start}")


def _latest_snapshot_path_in_dir(root: Path) -> Path:
    """Return the most recently modified snapshot in a directory."""
    if not root.exists() or not root.is_dir():
        raise RuntimeError(f"snapshot directory is not present: {root}")

    candidates = list(root.glob("*.sqlite"))
    if not candidates:
        raise RuntimeError(f"no snapshot files found in {root}")

    return max(candidates, key=lambda p: p.stat().st_mtime)


def _testing_api_base_url() -> str:
    """Return the testing API base URL."""
    return os.getenv("HAA_TESTING_API_URL", "http://127.0.0.1:8081").rstrip("/")


def _testing_flow_test_name() -> str:
    """Return the testing dataset name used by the validation flow."""
    return os.getenv("HAA_DAGSTER_TEST_NAME", "dagster-asset-flow")


def _testing_flow_snapshot_name() -> str:
    """Return the testing snapshot slug used by the validation flow."""
    return os.getenv("HAA_DAGSTER_SNAPSHOT_NAME", "dagster-asset-flow")


def _post_json(url: str, payload: dict) -> tuple[int, dict]:
    """POST JSON and return HTTP status with decoded JSON payload."""
    req = urllib.request.Request(
        url=url,
        data=json.dumps(payload).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            body = resp.read().decode("utf-8")
            try:
                parsed = json.loads(body) if body else {}
            except json.JSONDecodeError:
                parsed = {"error": body} if body else {}
            return resp.status, parsed
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8") if exc.fp is not None else ""
        try:
            parsed = json.loads(body) if body else {}
        except json.JSONDecodeError:
            parsed = {"error": body} if body else {}
        return exc.code, parsed


def _require_status(status: int, expected: int, step: str, payload: dict) -> None:
    """Raise a runtime error when an API call returns an unexpected status."""
    if status != expected:
        raise RuntimeError(f"{step} failed: expected {expected}, got {status}, payload={payload}")


def _summarize_snapshot(
    context: AssetExecutionContext, snapshot_path_fn, label: str
) -> MaterializeResult:
    """Read a snapshot DB and publish row-count metadata."""
    try:
        snapshot_path = snapshot_path_fn()
    except RuntimeError as exc:
        context.log.warning(str(exc))
        return MaterializeResult(metadata={"snapshot_missing": True})

    if label == "testing":
        context.log.info("using testing snapshot %s", snapshot_path)
    else:
        context.log.info("using snapshot %s", snapshot_path)

    with closing(sqlite3.connect(snapshot_path)) as conn:
        cur = conn.cursor()
        try:
            cur.execute("SELECT COUNT(*) FROM controls")
            controls_count = cur.fetchone()[0]
            cur.execute("SELECT COUNT(*) FROM aggregates")
            aggregates_count = cur.fetchone()[0]
        except (sqlite3.OperationalError, sqlite3.DatabaseError) as exc:
            message = f"failed snapshot query for {snapshot_path}: {exc}"
            context.log.exception(message)
            raise RuntimeError(message) from exc

    return MaterializeResult(
        metadata={
            "snapshot_path": str(snapshot_path),
            "controls_count": controls_count,
            "aggregates_count": aggregates_count,
        }
    )


@asset
def snapshot_summary(context: AssetExecutionContext) -> MaterializeResult:
    """Materialize summary metadata for the latest production snapshot."""
    return _summarize_snapshot(context, _latest_snapshot_path, "main")


@asset
def testing_snapshot_summary(context: AssetExecutionContext) -> MaterializeResult:
    """Materialize summary metadata for the latest testing snapshot."""
    return _summarize_snapshot(context, _latest_testing_snapshot_path, "testing")


@asset
def testing_api_snapshot_validation(context: AssetExecutionContext) -> MaterializeResult:
    """Exercise testing API flow and validate the exported snapshot contents."""
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

    with closing(sqlite3.connect(snapshot_path)) as conn:
        try:
            controls_count = conn.execute("SELECT COUNT(*) FROM controls").fetchone()[0]
            aggregates_count = conn.execute("SELECT COUNT(*) FROM aggregates").fetchone()[0]
        except (sqlite3.OperationalError, sqlite3.DatabaseError) as exc:
            message = f"failed snapshot query for {snapshot_path}: {exc}"
            # Testing-validation assets surface DB failures as metadata so CI can
            # report a structured failure reason without a hard Dagster crash.
            context.log.exception(message)
            return MaterializeResult(metadata={"snapshot_missing": True, "error": message})

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
    """Trigger snapshot processing when a newer snapshot file appears."""
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
