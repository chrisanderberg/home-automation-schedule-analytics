"""Tests for snapshot-path discovery helpers in analytics assets."""

import os
import sys
import tempfile
import time
import types
import unittest
from pathlib import Path

# Stub dagster to avoid importing the runtime stack in utility-focused tests.
dagster_stub = types.ModuleType("dagster")
dagster_stub.AssetExecutionContext = object
dagster_stub.MaterializeResult = object
dagster_stub.RunRequest = object
dagster_stub.SensorEvaluationContext = object
dagster_stub.SkipReason = object
dagster_stub.asset = lambda fn: fn
dagster_stub.sensor = lambda **_kwargs: (lambda fn: fn)
sys.modules["dagster"] = dagster_stub

from analytics.assets import _latest_snapshot_path_in_dir, _snapshot_root, _testing_snapshot_root
from analytics.tests._helpers import repository_root_for_test


class LatestSnapshotPathTests(unittest.TestCase):
    """Unit tests for selecting and resolving snapshot file paths."""

    def test_fixed_snapshot_dir_is_data_snapshots(self):
        """Snapshot root should resolve to <repo>/data/snapshots."""
        expected = _snapshot_path_root_for_test()
        self.assertEqual(_snapshot_root(), expected)

    def test_selects_newest_sqlite_file(self):
        """Latest helper should pick the file with the newest mtime."""
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            first = root / "snapshot-20260101-000000.sqlite"
            second = root / "snapshot-20260102-000000.sqlite"
            first.write_bytes(b"a")
            second.write_bytes(b"b")

            base = time.time()
            os.utime(first, (base, base))
            os.utime(second, (base + 1, base + 1))
            self.assertEqual(_latest_snapshot_path_in_dir(root), second)

    def test_fixed_testing_snapshot_dir_is_test_data_snapshots(self):
        """Testing snapshot root should resolve to <repo>/test-data/snapshots."""
        expected = repository_root_for_test() / "test-data" / "snapshots"
        self.assertEqual(_testing_snapshot_root(), expected)


def _snapshot_path_root_for_test() -> Path:
    """Return the expected production snapshot root for tests."""
    return repository_root_for_test() / "data" / "snapshots"

if __name__ == "__main__":
    unittest.main()
