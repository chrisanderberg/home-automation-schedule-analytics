# Analytics (Dagster)

## Setup
1. Create and activate a new conda env (any name).
2. From this directory, install deps with uv (inside conda, use `--system`):

```bash
uv pip install -e . --system
```

## Run
```bash
dagster dev
```

This should launch the Dagster UI and load the project definitions in `analytics`.
