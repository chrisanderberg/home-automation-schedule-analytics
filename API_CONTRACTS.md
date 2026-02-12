# API_CONTRACTS.md

## Canonical scope
This file is canonical for HTTP contract shape and endpoint behavior.

## Base topology
- Main API: `:8080`
- Testing API: `:8081`

Shared endpoints on both APIs:
- `POST /v1/controls`
- `POST /v1/holding-intervals`
- `POST /v1/transitions`
- `POST /v1/snapshots`

Testing-only endpoint:
- `POST /v1/reset`

## Control metadata upsert
Endpoint:
- `POST /v1/controls`

Required fields:
- `controlId`
- `controlType` (`discrete` or `slider`)
- `numStates`

Behavior:
- Upserts control metadata by `controlId`.
- `controlType` determines how `numStates` is interpreted:
  - `controlType == "discrete"`: `numStates` is required and must be `2..10`.
  - `controlType == "slider"`: `numStates` is treated as a quantization count (recommended `6`) and must be `2..10` in the current API.
- Invalid payloads are rejected with clear errors.

Examples:
- Discrete: `{ "controlId": "mode", "controlType": "discrete", "numStates": 3 }`
- Slider: `{ "controlId": "setpoint", "controlType": "slider", "numStates": 6 }`

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
- Self-transitions (`fromState == toState`) are rejected.
- Invalid payloads are discarded with clear logging.

## Snapshot export
Endpoint:
- `POST /v1/snapshots`

Behavior:
- Main API writes snapshots under `data/snapshots`.
- Testing API writes snapshots under `test-data/snapshots`.
- Testing snapshot exports overwrite same-name files.

## Testing API payload extensions
For all testing control/ingestion/snapshot requests:
- `testName` is required.

For testing snapshot requests:
- `snapshotName` is required.

Slug constraint for `testName` and `snapshotName`:
- `^[a-z0-9]+(?:-[a-z0-9]+)*$`

## Non-configurable runtime paths
- Main API payloads cannot override DB/snapshot runtime paths.
- Testing requests cannot write to main runtime data paths.
- Analytics/Dagster must consume snapshots only and must not read or mutate `data/data.sqlite` or `test-data/*-test-data.sqlite`.
