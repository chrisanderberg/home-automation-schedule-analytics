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

Snapshots are exported under `/aggregation/snapshot` with timestamped filenames.

## Analytics
From `/analytics`:
```bash
uv pip install -e .
export HAA_SNAPSHOT_DIR=/path/to/snapshots
# or omit HAA_SNAPSHOT_DIR to default to /aggregation/snapshot

dagster dev
```
