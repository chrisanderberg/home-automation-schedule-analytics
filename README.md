# Home Automation Analytics

Monorepo for a home automation aggregation service and analytics reporting.

## Structure
- `/aggregation`: Go aggregation service (ingestion, bucketing, storage, snapshots)
- `/analytics`: Dagster analytics project (reads snapshot SQLite files)

## Aggregation
Run tests from `/aggregation`:
```bash
go test ./...
```

The live aggregation DB defaults to `/aggregation/data/data.sqlite`.
Snapshots are exported under `/aggregation/data/snapshots` with timestamped filenames.

## Analytics
From `/analytics`:
```bash
uv pip install -e .
dagster dev
```
