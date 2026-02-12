# Aggregation Service Reviewer Guide

This package implements ingestion and aggregation for home automation control data.

## Data Flow

1. HTTP request enters main/testing API handlers in `api/`.
2. `ingest/` validates payloads, resolves control metadata/timezone, and splits:
   - by UTC quarter (`quarter/`)
   - by time-of-week buckets (`bucketing/`)
3. Counters are updated in dense blobs (`blob/`) and persisted via `storage/`.
4. `snapshot/` exports SQLite copies for Dagster consumption.

## Core Invariants

- Holding intervals are half-open: `[startTimeMs, endTimeMs)`.
- Day indexing is Monday=0 ... Sunday=6.
- Bucket size is fixed at 5 minutes (2016 buckets/week).
- All five clocks share the same bucket cardinality and are stored in parallel.
- Quarter partitioning is UTC calendar quarters only.
- Transition diagonal (`fromState == toState`) is invalid.

## Where To Review Specific Behavior

- Dense blob index layout: `blob/accessor.go`
- UTC/local/solar bucket mapping and interval splitting: `bucketing/`
- Quarter splitting logic: `quarter/quarter.go`
- Storage transaction behavior and upsert semantics: `storage/store.go`
- Ingestion orchestration (holding/transition): `ingest/`
- API request contracts and strict JSON handling: `api/`
- Snapshot copy mechanics and naming contracts: `snapshot/snapshot.go`

## Runtime Paths

- Main DB: `data/data.sqlite`
- Main snapshots: `data/snapshots/*.sqlite`
- Testing DBs: `test-data/<testName>-test-data.sqlite`
- Testing snapshots: `test-data/snapshots/<testName>-<snapshotName>-snapshot.sqlite`
