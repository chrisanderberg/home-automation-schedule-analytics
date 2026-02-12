# TESTING.md

## Canonical scope
This file is canonical for test commands, expected outcomes, and test-data conventions.

## Fast checks
Run from `aggregation`:

```bash
go test ./...
```

Expected result:
- All tests pass.

## Analytics checks
Run from `analytics`:

```bash
uv pip install -e . --system
dagster dev
```

Expected result:
- Dagster UI starts and project definitions load.

If any analytics Python command fails with missing executable/module errors (for example `dagster`, `ruff`, `pytest`, or import failures):
- Ask for project-specific environment activation steps first (e.g., conda, venv, or other local workflow).
- Avoid committing machine-specific environment details (such as personal conda paths) to tracked repo docs.
- Keep local environment specifics in local-only setup files or personal shell config.

## Analytics tests
Run from `analytics`:

```bash
PYTHONPATH=src python3 -m unittest discover -s tests -p 'test_*.py'
```

Expected result:
- All analytics unit tests pass.

## Analytics docstring lint
Run from repository root:

```bash
ruff check --config analytics/pyproject.toml --select D analytics/src
```

Expected result:
- No docstring violations in production analytics Python modules.

## Optional local pre-commit setup
Run from repository root:

```bash
python3 -m pip install pre-commit
pre-commit install
pre-commit run --all-files
```

Expected result:
- Pre-commit enforces analytics docstring lint before commit.

## Test-data naming conventions
All paths below are relative to the repository root.
- Testing DB path: `test-data/<testName>-test-data.sqlite`
- Testing snapshot path: `test-data/snapshots/<testName>-<snapshotName>-snapshot.sqlite`
- Slug format for `testName` and `snapshotName`: `^[a-z0-9]+(?:-[a-z0-9]+)*$`

## Notes
- Main runtime data and testing runtime data must remain isolated.
