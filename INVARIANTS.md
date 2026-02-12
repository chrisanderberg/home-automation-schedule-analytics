# INVARIANTS.md

## Canonical scope
This file is canonical for cross-cutting invariants that must hold across implementation.

## Time and bucket invariants
- Holding interval semantics: `[startTimeMs, endTimeMs)`.
- Bucket size: 5 minutes.
- Buckets per day: 288.
- Buckets per week: 2016.
- Day index mapping: Monday=0 ... Sunday=6.

## Clock invariants
- Exactly five clocks are computed in parallel:
  - UTC
  - Local time
  - Mean solar time
  - Apparent solar time
  - Unequal hours
- Clock order in blob groups is fixed and canonical.
- Undefined mapping for one clock drops only that clock’s contribution.

## Quarter invariants
- Quarter windows are UTC calendar quarters only.
- Intervals crossing quarter boundaries must be split.

## Transition invariants
- `fromState == toState` is invalid and must not be stored.
- Transition counts represent human-initiated corrections only.

## Dense blob invariants
- Canonical constants:
  - `B=2016`, `C=5`, `G=10080`, `N=numStates`
- Total stored values per aggregate blob:
  - `N^2 * G`
- Numeric encoding:
  - All values stored as little-endian `u64`.
- Index math:
  - `holdIndex(s,c,b) = (s*G) + (c*B) + b`
  - `offsetWithinFromBlock(from,to) = (to < from) ? to : (to - 1)`
  - `transGroupIndex(from,to) = from*(N-1) + offsetWithinFromBlock(from,to)`
  - `transIndex(from,to,c,b) = (N*G) + (transGroupIndex(from,to)*G) + (c*B) + b`

## Runtime path invariants
- Main DB path fixed at `data/data.sqlite`.
- Main snapshots under `data/snapshots`.
- Testing DB data isolated under `test-data`.
- Testing snapshots under `test-data/snapshots`.
- Dagster must not read or mutate live/testing DB files; it consumes snapshots only.
- Testing names use lowercase slug format only.
