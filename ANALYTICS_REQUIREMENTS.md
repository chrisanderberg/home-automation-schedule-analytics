# ANALYTICS_REQUIREMENTS.md

## Analytics requirements (canonical for analytics domain)

### Scope
- This file is canonical for analytics behavior, orchestration, and output contracts.
- Analytics is implemented in Dagster and consumes aggregation snapshot artifacts.
- Do not introduce semantics that contradict this file without explicit user approval.

### Analytics purpose
- Generate preference-oriented analytics from aggregated holding and transition statistics.
- Produce reproducible artifacts/report outputs for inspection in Dagster runs.
- Support cross-model comparison for each control/time bucket under each clock.

### Core hypothesis
For a given control and time-of-week (per clock), multiple automation models may have been active historically:
1. Holding time reflects what the automation model caused the system to do.
2. User-initiated transitions reflect what the user prefers (corrections).
3. If inference is correct, preference estimates should converge across models:
   occupancy may differ, but inferred preference (CTMC stationary distribution) should align for the same control/time bucket.

### Analytics input contract
- Input source is a SQLite snapshot file under `/aggregation/data/snapshots`.
- Analytics reads the newest `*.sqlite` by modification time.
- Analytics must not read from live DB paths.
- Missing snapshots should produce a clear, non-crashing Dagster skip/warning path.

### Required pipeline stages
1. Snapshot loading
- Read controls and aggregate blobs from the chosen snapshot.
- Validate schema presence and fail with explicit diagnostics when missing/invalid.

2. Sufficient-statistics decode
- Decode dense blob payload into holding and transition tensors using canonical index math/order from `AGGREGATION_REQUIREMENTS.md`.

3. KDE smoothing
- KDE smooths sparse weekly bucketed data cyclically across time-of-week buckets.
- KDE outputs smoothed holding times per state and smoothed transition counts per directed state pair.
- KDE is applied before CTMC estimation.

4. CTMC estimation
- Build a CTMC per `(controlId, modelId, quarterIndex, clock, bucket)`.
- For `i != j`, off-diagonal rate is proportional to transitions `i->j` divided by holding time in state `i`.
- Include stability handling for zero/near-zero holding-time regimes.

5. Stationary distribution
- Compute stationary distribution for each estimated CTMC.
- Treat stationary distribution as the preference estimate.
- Record convergence/quality metadata for review/debugging.

### Output contract
- Dagster runs must surface materialized artifacts/metadata sufficient to inspect:
  - selected snapshot path and timestamp
  - controls/aggregate row counts
  - KDE output presence/shape summary
  - CTMC/stationary output presence/shape summary
- Output schema details are versioned in code and validated by tests.

### Orchestration contract
- Snapshot detection sensor is the primary trigger for new-snapshot processing.
- Daily schedule is a backstop run to keep visibility fresh even if sensor events are missed.
- Analytics assets/jobs must be idempotent with respect to duplicate triggers for the same snapshot.

### Testing requirements
- Provide deterministic fixture snapshots for analytics tests.
- Include golden tests for:
  - blob decode correctness
  - KDE smoothing behavior on cyclic time-of-week data
  - CTMC/stationary estimation for known small systems
- Include one end-to-end Dagster test proving snapshot input -> analytics output artifact path.

### Reviewability/readability
- Non-obvious math and numerical safeguards must include reviewer-oriented comments describing intent and invariants.
- Constants and model dimensions must be named (avoid magic numbers).
