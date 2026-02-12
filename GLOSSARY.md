# GLOSSARY.md

## Canonical scope
This file is canonical for shared domain terminology.

## Terms
- Control: A discrete state variable in home automation.
- Discrete/radio control: Control with `2..10` valid states.
- Slider control: Control discretized to `6` states.
- Model: Automation model associated with a control’s behavior.
- Holding interval: Time spent in a specific state over `[startTimeMs, endTimeMs)`.
- Transition event: User-initiated correction from one state to another.
- Bucket: 5-minute time-of-week bin (2016 per week).
- Bucket index (`b`): `dayIndex*288 + bucketWithinDay`, where `b in [0,2015]`.
- Clock: One of five coordinate systems (UTC, Local, Mean Solar, Apparent Solar, Unequal Hours).
- Undefined clock mapping: A timestamp/interval that cannot map for a specific clock (drop only that clock).
- Quarter window: UTC calendar quarter used for aggregate partitioning.
- Quarter index: `(utcYear - 1970) * 4 + (quarterNumber - 1)`.
- Dense blob: `u64` little-endian payload storing holding times and transition counts.
- Snapshot: SQLite copy exported for analytics consumption.
- Main API: External application API on port `8080`.
- Testing API: Internal/testing API on port `8081` with isolated test data and reset support.
- `testName`: Required slug identifying a test data namespace.
- `snapshotName`: Required slug for testing snapshot exports.
