# AGENTS.md

## Instructions (read first)

### Authority and scope
- This file is the single source of truth for:
  1) how to work in this repo,
  2) the requirements, and
  3) the milestone plan.
- Requirements in this file may be changed only with explicit user approval. It is okay to ask for changes, but do not change requirements without asking first.
- Do not introduce new semantics that contradict the Requirements section below.
- Keep changes tightly scoped to the current milestone.

### Milestone execution rule (two-step)
Each milestone must be implemented in two steps:

1) Scaffolding + tests
- Create file structure and stubs.
- Add tests (and/or golden fixtures) that define correct behavior.
- Ensure `go test ./...` runs and tests compile (tests may fail if stubs are TODO).

2) Implementation
- Fill in TODOs until tests pass.
- Ensure `go test ./...` passes before completing the milestone.

### Assumptions and TBD handling
- If something is not specified, do not guess silently.
- Prefer parameterization when possible.
- Record any necessary assumptions in the “Assumptions / Decisions log” section at the end
  of this file.

### Output format expectations (for coding agents)
For any milestone implementation, include:
- files added/changed
- tests added/changed
- commands to run to verify
- assumptions made (if any)

---

## Commands (copy/paste)

These commands must work once the repo is bootstrapped (run from `/aggregation`):

```bash
go test ./...
```

Additional run/build commands will be added once the repo layout is established.

---

## Requirements (single source of truth)

### Repo purpose and scope
- This repository is a monorepo with two components:
  - A Go aggregation service that ingests control measurement events (holding intervals and
    transitions) and stores dense, bucketed aggregates in SQLite.
  - A Dagster analytics project that reads SQLite snapshot files and generates analytics
    outputs and reports. Outputs include KDE, CTMC, and stationary distribution estimates.
    The Dagster web UI is used to view runs and report artifacts.
- Out of scope:
  - custom user control UI/control panel
  - changing canonical clock set, bucket definitions, or blob layout
  - production auth/security/scaling

### High-level model
Controls are discrete state variables.

- Discrete/radio controls: N states, where 2 <= N <= 10
- Slider controls: N = 6 (discretized slider states)

Controls are associated with automation models. Both automation and humans can
change control state. Only human-initiated transitions are counted as transitions
for analytics.

This repo’s aggregation service assumes upstream only sends countable (human)
transitions to the transition ingestion endpoint.

### Five clocks (always computed in parallel)
The system uses exactly five clocks, always computed and stored in parallel:

0) UTC
1) Local time
2) Mean solar time
3) Apparent solar time
4) Unequal hours

Clocks are treated as an experiment (schedule A/B test): analytics compares how
preference estimates correlate under different time coordinate systems.

Service-wide clock configuration:
- The aggregation service uses a single global configuration for timezone and location.
- Timezone is required for Local time bucketing (including DST handling).
- Latitude/longitude are required for solar clocks (mean solar, apparent solar, unequal hours).
- Configuration is provided by a config file and may be overridden by environment variables.

### Time-of-week bucketing
- 5-minute buckets
- 288 buckets/day (24 * 12)
- 2016 buckets/week (7 * 288)
- Buckets are indexed per clock.

Day-of-week indexing convention:
- Monday = 0
- Tuesday = 1
- Wednesday = 2
- Thursday = 3
- Friday = 4
- Saturday = 5
- Sunday = 6

Bucket index `b` in 0..2015:

- `bucketWithinDay = hour * 12 + floor(minute / 5)` (0..287)
- `b = dayIndex * 288 + bucketWithinDay` (0..2015)

### Measurement semantics (aggregation inputs)
Two fundamental operations are ingested:

1) Holding interval (time in state)
- Input: controlId, modelId, state, startTimeMs, endTimeMs
- Time representation: UTC epoch milliseconds (integers)
- Interval semantics: half-open [startTimeMs, endTimeMs)
- Holding time is measured in real elapsed milliseconds.
- The holding interval is split across all overlapped time-of-week buckets, per
  clock, and accumulated.

2) Transition event (user correction)
- Input: controlId, modelId, fromState, toState, timestampMs
- Counted in the time-of-week bucket containing timestampMs, per clock.
- Self-transitions (fromState == toState) are rejected/not stored.

Data integrity posture:
- If integrity fails (invalid timestamps, invalid states, missing control
  metadata, etc.), discard the input and log a clear reason.

Undefined time-of-day cases:
- If a clock mapping is undefined (e.g., unequal hours when no sunrise/sunset),
  do not count data for that clock only; still count other clocks when defined.

### Quarter windows (UTC calendar quarters)
Quarter windows are UTC calendar quarters:

- Q1: January–March
- Q2: April–June
- Q3: July–September
- Q4: October–December

Quarters are variable-length and may include leap days.

Quarter windows are independent of the five clocks.

A suggested integer representation:

- `quarterIndex = (utcYear - 1970) * 4 + (quarterNumber - 1)`

If an ingested holding interval crosses a quarter boundary, it must be split and
applied to multiple quarter windows.

### Storage model (SQLite) — conceptual
The aggregation service stores:

1) Control metadata:
- controlId
- control type (discrete vs slider)
- numStates (2..10; slider typically 6)
- optional state labels

2) Aggregated sufficient statistics keyed by:
- controlId
- modelId
- quarterIndex (UTC calendar quarter)
- payload: dense numeric blob described below

SQLite is the canonical storage format for aggregates.

### Dense blob layout (canonical)
Constants:
- B = 2016 buckets/week
- C = 5 clocks
- G = B * C = 10080 values per “bucket group”
- N = numStates

Clock ordering within each group of 10080:
0) UTC (2016)
1) Local (2016)
2) Mean solar (2016)
3) Apparent solar (2016)
4) Unequal hours (2016)

Blob structure for one (controlId, modelId, quarterIndex):

1) Holding times (milliseconds), grouped by state:
- state 0 holding: G values
- state 1 holding: G values
- ...
- state N-1 holding: G values

2) Transition counts, grouped by (fromState, toState) excluding diagonal:
- (0→1), (0→2), ..., (0→N-1)
- (1→0), (1→2), ..., (1→N-1)
- ...
- (N-1→0), (N-1→1), ..., (N-1→N-2)

Total stored values:
- (N + N*(N-1)) * G = N^2 * G

Index math (zero-based):

- `holdIndex(s,c,b) = (s*G) + (c*B) + b`

Transition group indexing:
- `offsetWithinFromBlock(from,to) = (to < from) ? to : (to - 1)`
- `transGroupIndex(from,to) = from*(N-1) + offsetWithinFromBlock(from,to)`

- `transIndex(from,to,c,b) = (N*G) + (transGroupIndex(from,to)*G) + (c*B) + b`

Numeric encoding decision:
- Store all blob values as unsigned 64-bit integers (u64) in little-endian.
  - holding times: elapsed milliseconds
  - transition counts: integer increments

### Dagster snapshot rule (analytics input)
Dagster analytics must read from a SQLite snapshot file, not the live DB.

A daily cadence (about once per day) is the default.

Snapshot discovery convention:
- Dagster reads from the newest `*.sqlite` file in the directory specified by
  `HAA_SNAPSHOT_DIR` (latest by modification time).

Snapshot export convention:
- Aggregation snapshots are written under `/aggregation/snapshot`.
- Snapshot filenames include a timestamp (date/time) to enable ordering.

### Hypothesis (analytics intent)
For a given control and time-of-week (per clock), multiple automation models may
have been active historically. The hypothesis is:
1) Holding time reflects what the automation model caused the system to do.
2) User-initiated transitions reflect what the user prefers (corrections).
3) If inference is correct, preference estimates should converge across models:
   raw occupancy may differ, but inferred preference (via CTMC stationary
   distribution) should align across models for the same control/time bucket.

### KDE (smoothing)
- KDE smooths sparse bucketed data across nearby time-of-week buckets (cyclic).
- KDE produces smoothed sufficient statistics:
  - smoothed holding times per state
  - smoothed user-transition counts between states
- KDE is applied before CTMC estimation.

### CTMC (preference estimation)
- Build a CTMC per control and per query time (clock, time-of-week).
- Rates for i != j are proportional to user transitions i->j divided by holding
  time in state i.
- The stationary distribution of the CTMC is treated as the preference estimate.

---

## Plan (milestones and ordering)

Milestones are implemented using the two-step rule in Instructions.

1. Milestone 0 — Repo bootstrap + unit test harness
2. Milestone 1 — Dense blob accessors + invariants tests
3. Milestone 2 — SQLite schema + persistence for controls and aggregates
4. Milestone 3 — UTC and Local clock bucketing + interval splitting
5. Milestone 4 — Quarter splitting (UTC calendar quarters)
6. Milestone 5 — Holding ingestion end-to-end (UTC + Local)
7. Milestone 6 — Transition ingestion end-to-end (UTC + Local)
8. Milestone 7 — Solar clocks (mean solar, apparent solar, unequal hours)
9. Milestone 8 — Snapshot export for Dagster (daily input artifact)
10. Milestone 9 — Dagster project scaffold + daily run
11. Milestone 10 — REST API scaffold + ingestion endpoints

---

## Assumptions / Decisions log

When new assumptions are made during implementation, append entries here:

- Date:
- Milestone:
- Assumption:
- Why needed:
- Impact/risk:
- Resolution (TBD / decided / update Requirements section above):
- Date: 2026-02-10
- Milestone: 0
- Assumption: Go module path set to `home-automation-analytics/aggregation`.
- Why needed: `go.mod` requires a module path to initialize the Go project.
- Impact/risk: May need to change if a canonical VCS import path is desired.
- Resolution (TBD / decided / update Requirements section above): TBD
- Date: 2026-02-10
- Milestone: 2
- Assumption: `controls.state_labels` stored as JSON-encoded string when present.
- Why needed: Requirements define optional labels but not an encoding.
- Impact/risk: Changing encoding later requires migration.
- Resolution (TBD / decided / update Requirements section above): TBD
