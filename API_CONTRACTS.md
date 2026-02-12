# API_CONTRACTS.md

## Canonical scope
This file is canonical for HTTP contract shape and endpoint behavior.

## Base topology
- Main API: `:8080`
- Testing API: `:8081`

Shared endpoints on both APIs:
- `POST /v1/holding-intervals`
- `POST /v1/transitions`
- `POST /v1/snapshots`

Testing-only endpoint:
- `POST /v1/reset`

## Holding interval ingestion
Endpoint:
- `POST /v1/holding-intervals`

Required fields:
- `controlId`
- `modelId`
- `state`
- `startTimeMs`
- `endTimeMs`

Behavior:
- Interval is half-open: `[startTimeMs, endTimeMs)`.
- Interval is bucket-split per defined clock.
- Interval is quarter-split on UTC calendar quarter boundaries.
- Invalid payloads are discarded with clear logging.

## Transition ingestion
Endpoint:
- `POST /v1/transitions`

Required fields:
- `controlId`
- `modelId`
- `fromState`
- `toState`
- `timestampMs`

Behavior:
- Counted in bucket containing `timestampMs` per defined clock.
- Self transitions (`fromState == toState`) are rejected.
- Invalid payloads are discarded with clear logging.

## Snapshot export
Endpoint:
- `POST /v1/snapshots`

Behavior:
- Main API writes snapshot under `/aggregation/data/snapshots`.
- Testing API writes under `/aggregation/test-data/snapshots`.
- Testing snapshot exports overwrite same-name files.

## Testing API payload extensions
For all testing ingestion/snapshot requests:
- `testName` is required.

For testing snapshot requests:
- `snapshotName` is required.

Slug constraint for both:
- `^[a-z0-9]+(?:-[a-z0-9]+)*$`

## Non-configurable runtime paths
- Main API payloads cannot override DB/snapshot runtime paths.
- Testing requests cannot write to main runtime data paths.
