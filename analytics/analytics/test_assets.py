import os
import sys
import tempfile
import types
import unittest
from pathlib import Path

dagster_stub = types.ModuleType("dagster")
dagster_stub.AssetExecutionContext = object
dagster_stub.MaterializeResult = object
dagster_stub.RunRequest = object
dagster_stub.SensorEvaluationContext = object
dagster_stub.SkipReason = object
dagster_stub.asset = lambda fn: fn
dagster_stub.sensor = lambda **_kwargs: (lambda fn: fn)
sys.modules.setdefault("dagster", dagster_stub)

from analytics.assets import _latest_snapshot_path_in_dir


class LatestSnapshotPathTests(unittest.TestCase):
    def test_fixed_snapshot_dir_is_data_snapshots(self):
        expected = Path(__file__).resolve().parents[2] / "aggregation" / "data" / "snapshots"
        self.assertEqual(_snapshot_path_root_for_test(), expected)

    def test_selects_newest_sqlite_file(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            first = root / "snapshot-20260101-000000.sqlite"
            second = root / "snapshot-20260102-000000.sqlite"
            first.write_bytes(b"a")
            second.write_bytes(b"b")

            os.utime(first, (1, 1))
            os.utime(second, (2, 2))
            self.assertEqual(_latest_snapshot_path_in_dir(root), second)


def _snapshot_path_root_for_test() -> Path:
    return Path(__file__).resolve().parents[2] / "aggregation" / "data" / "snapshots"

if __name__ == "__main__":
    unittest.main()
