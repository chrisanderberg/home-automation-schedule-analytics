# AGGREGATION_REQUIREMENTS.md

## Aggregation requirements (canonical for aggregation domain)

### Scope
- This file is canonical for aggregation service behavior, storage, and API/runtime contracts.
- Do not introduce semantics that contradict this file without explicit user approval.

### Five clocks (always computed in parallel)
The system uses exactly five clocks, always computed and stored in parallel:

0) UTC
1) Local time
2) Mean solar time
3) Apparent solar time
4) Unequal hours

Service-wide clock configuration:
- The aggregation service uses a single global configuration for timezone and location.
- Timezone is required for Local time bucketing (including DST handling).
- Latitude/longitude are required for solar clocks (mean solar, apparent solar, unequal hours).
- Configuration is provided by a config file and may be overridden by environment variables.

### Time-of-week bucketing
- 5-minute buckets
- 288 buckets/day (`24 * 12`)
- 2016 buckets/week (`7 * 288`)
- Buckets are indexed per clock.

Day-of-week indexing convention:
- Monday = 0
- Tuesday = 1
- Wednesday = 2
- Thursday = 3
- Friday = 4
- Saturday = 5
- Sunday = 6

Bucket index `b` in `0..2015`:
- `bucketWithinDay = hour * 12 + floor(minute / 5)` (`0..287`)
- `b = dayIndex * 288 + bucketWithinDay` (`0..2015`)

### Measurement semantics (aggregation inputs)
Two fundamental operations are ingested:

1) Holding interval (time in state)
- Input: `controlId`, `modelId`, `state`, `startTimeMs`, `endTimeMs`
- Time representation: UTC epoch milliseconds (integers)
- Interval semantics: half-open `[startTimeMs, endTimeMs)`
- Holding time is measured in real elapsed milliseconds.
- Interval is split across all overlapped time-of-week buckets, per clock, and accumulated.

2) Transition event (user correction)
- Input: `controlId`, `modelId`, `fromState`, `toState`, `timestampMs`
- Counted in the time-of-week bucket containing `timestampMs`, per clock.
- Self-transitions (`fromState == toState`) are rejected/not stored.

Data integrity posture:
- If integrity fails (invalid timestamps, invalid states, missing control metadata, etc.), discard input and log a clear reason.

Undefined time-of-day cases:
- If a clock mapping is undefined (e.g., unequal hours when no sunrise/sunset), do not count data for that clock only; still count other clocks when defined.

### Quarter windows (UTC calendar quarters)
Quarter windows are UTC calendar quarters:
- Q1: January-March
- Q2: April-June
- Q3: July-September
- Q4: October-December

Quarters are variable-length and may include leap days.
Quarter windows are independent of the five clocks.

Suggested integer representation:
- `quarterIndex = (utcYear - 1970) * 4 + (quarterNumber - 1)`

If an ingested holding interval crosses a quarter boundary, split and apply to multiple quarter windows.

### Storage model (SQLite) - conceptual
The aggregation service stores:

1) Control metadata:
- `controlId`
- control type (discrete vs slider)
- `numStates` (`2..10`; slider typically `6`)
- optional state labels

2) Aggregated sufficient statistics keyed by:
- `controlId`
- `modelId`
- `quarterIndex` (UTC calendar quarter)
- payload: dense numeric blob described below

SQLite is the canonical storage format for aggregates.

Runtime file locations:
- Live DB path fixed at `/aggregation/data/data.sqlite`.
- Snapshot files are runtime artifacts under `/aggregation/data/snapshots`.
- `/aggregation/snapshot` is code-only (snapshot logic), not a destination for generated files.
- Runtime file locations are fixed and not user-configurable via API payloads, flags, or env vars.
- Testing runtime data is isolated under `/aggregation/test-data`.
- Testing DB files are named `/aggregation/test-data/<testName>-test-data.sqlite`.
- Testing snapshot files are named `/aggregation/test-data/snapshots/<testName>-<snapshotName>-snapshot.sqlite`.
- `testName` and `snapshotName` use lowercase slugs only: `^[a-z0-9]+(?:-[a-z0-9]+)*$`.

### Dense blob layout (canonical)
Constants:
- `B = 2016` buckets/week
- `C = 5` clocks
- `G = B * C = 10080` values per bucket group
- `N = numStates`

Clock ordering within each group of 10080:
0) UTC (2016)
1) Local (2016)
2) Mean solar (2016)
3) Apparent solar (2016)
4) Unequal hours (2016)

Blob structure for one (`controlId`, `modelId`, `quarterIndex`):

1) Holding times (milliseconds), grouped by state:
- state 0 holding: `G` values
- state 1 holding: `G` values
- ...
- state `N-1` holding: `G` values

2) Transition counts, grouped by (`fromState`, `toState`) excluding diagonal:
- `(0->1), (0->2), ..., (0->N-1)`
- `(1->0), (1->2), ..., (1->N-1)`
- ...
- `(N-1->0), (N-1->1), ..., (N-1->N-2)`

Total stored values:
- `(N + N*(N-1)) * G = N^2 * G`

Index math (zero-based):
- `holdIndex(s,c,b) = (s*G) + (c*B) + b`
- `offsetWithinFromBlock(from,to) = (to < from) ? to : (to - 1)`
- `transGroupIndex(from,to) = from*(N-1) + offsetWithinFromBlock(from,to)`
- `transIndex(from,to,c,b) = (N*G) + (transGroupIndex(from,to)*G) + (c*B) + b`

Numeric encoding decision:
- Store all blob values as unsigned 64-bit integers (`u64`) in little-endian.
  - holding times: elapsed milliseconds
  - transition counts: integer increments

### Snapshot export contract (analytics input artifact)
- Analytics must read from a snapshot SQLite file, not the live DB.
- Daily cadence (about once per day) is the default.

Snapshot discovery/export convention:
- Analytics reads newest `*.sqlite` in `/aggregation/data/snapshots` (latest by modification time).
- Snapshots are written under `/aggregation/data/snapshots`.
- Snapshot filenames include a timestamp for ordering.
- Snapshot filenames keep the `snapshot` prefix followed by a timestamp.

Testing snapshot convention:
- Testing snapshots are on-demand only (not scheduled).
- Testing snapshot exports overwrite existing files with the same `<testName>-<snapshotName>-snapshot.sqlite` name.

### API topology (aggregation service)
- One binary serves two HTTP APIs in parallel:
  - Main API on port `8080` (application layer)
  - Testing API on port `8081` (internal/testing only)
- Main and testing APIs share endpoint paths for ingestion and snapshots:
  - `POST /v1/holding-intervals`
  - `POST /v1/transitions`
  - `POST /v1/snapshots`
- Main and testing APIs must use the same aggregation logic implementation for holding/transition ingestion.
- Testing API adds `POST /v1/reset` (testing-only). No reset endpoint exists on the main API.
- Main API payloads do not accept path/database overrides.
- Testing API payloads include `testName` for all ingestion/snapshot requests, and include `snapshotName` for snapshot requests.
- Testing ingestion appends/aggregates into existing test DB files (no implicit reset).
- Testing data paths must never touch main data paths.

### Code readability and reviewer UX (aggregation code)
- Public functions and non-trivial internal helpers must include concise doc comments.
- Comments should explain intent/invariants ("why"), not obvious syntax.
- Non-obvious code paths (time boundary math, index math, transaction semantics, SQL generation, API validation) require short inline rationale comments.
- Avoid magic numbers when domain constants exist; use named constants for clock indices, bucket dimensions, status codes, and other identifiers.
- Prefer small focused helpers when one function mixes validation, transformation/splitting, persistence, and response shaping.
