# ARCHITECTURE.md

## Canonical scope
This file is canonical for system structure and component interaction.

## Repository architecture
- Monorepo with two components:
  - `/aggregation`: Go aggregation service (ingestion, bucketing, storage, snapshots)
  - `/analytics`: Dagster analytics project (reads snapshot SQLite files)

## End-to-end data flow
1. HTTP request enters main/testing API handlers in `/aggregation/api`.
2. `/aggregation/ingest` validates payloads and orchestrates updates.
3. Holding intervals are split:
   - by UTC quarter via `/aggregation/quarter`
   - by time-of-week buckets via `/aggregation/bucketing`
4. Aggregates are updated in dense blobs via `/aggregation/blob`.
5. Blobs and control metadata are persisted via `/aggregation/storage`.
6. `/aggregation/snapshot` exports SQLite snapshots for Dagster.
7. `/analytics` consumes newest snapshot from `data/snapshots`.
8. Analytics decodes dense blobs into sufficient statistics.
9. Analytics applies KDE smoothing across cyclic time-of-week buckets.
10. Analytics estimates CTMC rates and stationary distributions per control/time slice.
11. Dagster materializes analytics metadata/artifacts for run inspection.

## Requirements map
- Shared guardrails: `REQUIREMENTS.md`
- Aggregation behavior: `AGGREGATION_REQUIREMENTS.md`
- Analytics behavior: `ANALYTICS_REQUIREMENTS.md`

## Runtime topology
- One binary serves two APIs:
  - Main API on `:8080`
  - Testing API on `:8081`
- Main and testing APIs share ingestion/snapshot endpoint paths.
- Testing API adds `POST /v1/reset` only.

## Runtime storage paths
- Main DB: `data/data.sqlite`
- Main snapshots: `data/snapshots/*.sqlite`
- Testing DBs: `test-data/<testName>-test-data.sqlite`
- Testing snapshots: `test-data/snapshots/<testName>-<snapshotName>-snapshot.sqlite`
