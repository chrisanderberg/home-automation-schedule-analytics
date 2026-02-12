"""Dagster entrypoint for local `dagster dev` in a src-layout project."""

from pathlib import Path
import sys

_SRC = Path(__file__).resolve().parent / "src"
if str(_SRC) not in sys.path:
    sys.path.insert(0, str(_SRC))

from analytics.definitions import definitions  # noqa: E402

