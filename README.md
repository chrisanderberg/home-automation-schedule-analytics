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
The binary serves two APIs:
- Main API on `:8080` (fixed runtime paths under `data/`).
- Testing API on `:8081` (isolated runtime paths under `test-data/`).

Testing API payloads include `testName` (slug), and snapshot payloads include
`snapshotName` (slug). Testing DB/snapshot files are named:
- `test-data/<testName>-test-data.sqlite`
- `test-data/snapshots/<testName>-<snapshotName>-snapshot.sqlite`

## Analytics
From `/analytics`:
```bash
uv pip install -e .
dagster dev
```
