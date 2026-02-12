from pathlib import Path


def repository_root_for_test() -> Path:
    """Resolve the monorepo root by walking parent directories."""
    start = Path(__file__).resolve()
    for parent in start.parents:
        if (parent / "aggregation").is_dir():
            return parent
    raise AssertionError(f"repository root not found from {start}")
