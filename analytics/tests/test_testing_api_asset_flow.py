"""Integration test for testing-API snapshot export materialized as an asset."""

import json
import os
import shutil
import socket
import sqlite3
import subprocess
import time
import unittest
import urllib.error
import urllib.request
from contextlib import closing
from pathlib import Path

from analytics.tests._helpers import repository_root_for_test

try:
    from dagster import MaterializeResult, asset, materialize
except Exception:  # pragma: no cover - environment-dependent import
    MaterializeResult = None
    asset = None
    materialize = None


def _pick_free_port() -> int:
    """Pick an ephemeral localhost TCP port for test server startup."""
    # Known TOCTOU tradeoff: port is released after bind() and may be claimed
    # before aggregationd starts. Acceptable for this integration test.
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def _post_json(url: str, payload: dict) -> tuple[int, dict]:
    """POST JSON and return status code plus decoded JSON response."""
    req = urllib.request.Request(
        url=url,
        data=json.dumps(payload).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            body = resp.read().decode("utf-8")
            decoded = json.loads(body) if body else {}
            return resp.status, decoded
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8") if exc.fp is not None else ""
        try:
            decoded = json.loads(body) if body else {}
        except json.JSONDecodeError:
            decoded = {"error": body} if body else {}
        return exc.code, decoded


def _wait_for_health(base_url: str, timeout_seconds: float = 10.0) -> None:
    """Wait until the service health endpoint responds with HTTP 200."""
    deadline = time.time() + timeout_seconds
    last_error = None
    while time.time() < deadline:
        try:
            with urllib.request.urlopen(f"{base_url}/v1/health", timeout=2) as resp:
                if resp.status == 200:
                    return
        except (urllib.error.URLError, TimeoutError) as exc:
            last_error = exc
        time.sleep(0.1)
    raise AssertionError(f"service did not become healthy at {base_url}: {last_error}")


def _seed_test_control(db_path: Path) -> None:
    """Create minimal schema and seed one control row for test setup."""
    db_path.parent.mkdir(parents=True, exist_ok=True)
    with sqlite3.connect(db_path) as conn:
        conn.execute(
            """
            CREATE TABLE IF NOT EXISTS controls (
                control_id TEXT PRIMARY KEY,
                control_type TEXT NOT NULL,
                num_states INTEGER NOT NULL,
                state_labels TEXT
            );
            """
        )
        conn.execute(
            """
            CREATE TABLE IF NOT EXISTS aggregates (
                control_id TEXT NOT NULL,
                model_id TEXT NOT NULL,
                quarter_index INTEGER NOT NULL,
                blob BLOB NOT NULL,
                PRIMARY KEY (control_id, model_id, quarter_index)
            );
            """
        )
        conn.execute(
            """
            INSERT OR REPLACE INTO controls (control_id, control_type, num_states, state_labels)
            VALUES (?, ?, ?, ?);
            """,
            ("c1", "discrete", 2, None),
        )
        conn.commit()


@unittest.skipUnless(materialize is not None, "dagster is not available in this environment")
@unittest.skipUnless(shutil.which("go"), "go toolchain is not available")
class TestingAPIAssetFlowTests(unittest.TestCase):
    """Exercises testing API flow and validates resulting snapshot metadata."""

    def test_testing_api_snapshot_validated_as_asset(self):
        """Start aggregationd, export a test snapshot, and materialize validation."""
        repo_root = repository_root_for_test()
        aggregation_dir = repo_root / "aggregation"
        main_port = _pick_free_port()
        test_port = _pick_free_port()
        while test_port == main_port:
            test_port = _pick_free_port()
        main_addr = f"127.0.0.1:{main_port}"
        test_addr = f"127.0.0.1:{test_port}"
        testing_api_url = f"http://{test_addr}"

        proc = subprocess.Popen(
            [
                "go",
                "run",
                "./cmd/aggregationd",
                "-addr",
                main_addr,
                "-test-addr",
                test_addr,
                "-tz",
                "UTC",
            ],
            cwd=aggregation_dir,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )
        try:
            _wait_for_health(testing_api_url)

            test_name = f"asset-flow-{os.getpid()}-{test_port}"
            snapshot_name = "contains-control"
            db_path = repo_root / "test-data" / f"{test_name}-test-data.sqlite"
            snapshot_path = repo_root / "test-data" / "snapshots" / f"{test_name}-{snapshot_name}-snapshot.sqlite"
            try:
                _seed_test_control(db_path)

                status, _ = _post_json(
                    f"{testing_api_url}/v1/holding-intervals",
                    {
                        "testName": test_name,
                        "controlId": "c1",
                        "modelId": "m1",
                        "state": 1,
                        "startTimeMs": 1578268800000,
                        "endTimeMs": 1578269100000,
                    },
                )
                self.assertEqual(status, 202)

                status, _ = _post_json(
                    f"{testing_api_url}/v1/snapshots",
                    {"testName": test_name, "snapshotName": snapshot_name},
                )
                self.assertEqual(status, 200)

                self.assertTrue(snapshot_path.exists(), f"missing snapshot at {snapshot_path}")

                @asset(name="exported_testing_snapshot")
                def exported_testing_snapshot():
                    with closing(sqlite3.connect(snapshot_path)) as conn:
                        controls_count = conn.execute("SELECT COUNT(*) FROM controls").fetchone()[0]
                        aggregates_count = conn.execute("SELECT COUNT(*) FROM aggregates").fetchone()[0]

                    if controls_count < 1:
                        raise RuntimeError("snapshot has no controls")
                    return MaterializeResult(
                        metadata={
                            "snapshot_path": str(snapshot_path),
                            "controls_count": controls_count,
                            "aggregates_count": aggregates_count,
                        }
                    )

                result = materialize([exported_testing_snapshot])
                self.assertTrue(result.success)
            finally:
                if snapshot_path.exists():
                    snapshot_path.unlink()
                snapshots_dir = snapshot_path.parent
                try:
                    snapshots_dir.rmdir()
                except OSError:
                    pass
                try:
                    snapshots_dir.parent.rmdir()
                except OSError:
                    pass
                for path in (db_path, Path(str(db_path) + "-wal"), Path(str(db_path) + "-shm")):
                    if path.exists():
                        path.unlink()
        finally:
            proc.terminate()
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait(timeout=5)
            if proc.returncode not in (0, -15):
                stdout_data = proc.stdout.read() if proc.stdout is not None else ""
                stderr_data = proc.stderr.read() if proc.stderr is not None else ""
                self.fail(
                    "aggregationd exited unexpectedly with "
                    f"code {proc.returncode}\nstdout:\n{stdout_data}\nstderr:\n{stderr_data}"
                )


if __name__ == "__main__":
    unittest.main()
