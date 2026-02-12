# REQUIREMENTS.md

## Requirements (single source of truth index)

### Change control
- Requirements may be changed only with explicit user approval.
- It is okay to ask for requirement changes, but do not change requirements without asking first.

### Canonical requirements split
- Aggregation requirements are canonical in `AGGREGATION_REQUIREMENTS.md`.
- Analytics requirements are canonical in `ANALYTICS_REQUIREMENTS.md`.
- This file is the canonical umbrella index for cross-cutting scope and shared guardrails.

### Monorepo purpose and scope
- This repository is a monorepo with two components:
  - A Go aggregation service that ingests control measurement events and stores dense, bucketed aggregates in SQLite.
  - A Dagster analytics project that reads aggregation snapshot SQLite files and produces analytics outputs/reports.

### Out of scope
- custom user control UI/control panel
- changing canonical clock set, bucket definitions, or blob layout
- production auth/security/scaling

### Shared domain model (cross-cutting)
- Controls are discrete state variables.
  - Discrete/radio controls: `2 <= N <= 10`
  - Slider controls: `N = 6` (discretized slider states)
- Controls are associated with automation models.
- Both automation and humans can change control state.
- Only human-initiated transitions are countable transitions for analytics.
- Aggregation ingestion assumes upstream only sends countable (human) transitions to the transition endpoint.

### Shared clocks/buckets/quarters invariants
- Exactly five clocks are canonical and must remain unchanged:
  - UTC
  - Local time
  - Mean solar time
  - Apparent solar time
  - Unequal hours
- Time-of-week bucketing is canonical:
  - 5-minute buckets
  - 288 buckets/day
  - 2016 buckets/week
  - day indexing Monday=0 through Sunday=6
- Quarter windows are UTC calendar quarters.

### Shared storage/snapshot contract
- SQLite is the canonical aggregate storage format.
- Analytics consumes snapshot SQLite files, not the live DB.
- Dagster must not read or mutate live aggregation DB files (`data/data.sqlite`, `test-data/*-test-data.sqlite`).
- Runtime path conventions and API contract details are defined in:
  - `AGGREGATION_REQUIREMENTS.md`
  - `API_CONTRACTS.md`
  - `ARCHITECTURE.md`

### References
- Aggregation requirements: `AGGREGATION_REQUIREMENTS.md`
- Analytics requirements: `ANALYTICS_REQUIREMENTS.md`
- Cross-cutting invariants: `INVARIANTS.md`
