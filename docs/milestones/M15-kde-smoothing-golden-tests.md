# Milestone 15 - KDE smoothing implementation + golden tests

## Canonical scope
This file is canonical for Milestone 15 execution details.

## Objective
- Implement cyclic time-of-week KDE smoothing over decoded sufficient statistics.

## Source of truth references
- Requirements umbrella:
  - ../../REQUIREMENTS.md
- Aggregation requirements:
  - ../../AGGREGATION_REQUIREMENTS.md
- Analytics requirements:
  - ../../ANALYTICS_REQUIREMENTS.md
- Plan order:
  - ../../PLAN.md
- Process and DoD:
  - ../../AGENTS.md

## Acceptance checklist
- [ ] Scaffolding + tests step complete.
- [ ] Golden fixtures for cyclic KDE behavior are added.
- [ ] Edge cases (week wraparound, sparse buckets, all-zero windows) are covered.
- [ ] Implementation step complete.
- [ ] Analytics tests pass.
- [ ] No unapproved TODO sentinels in production code.

## Non-goals
- [ ] No CTMC/stationary distribution estimator implementation in this milestone.

## Notes
- Preserve deterministic outputs for golden tests (fixed parameters, fixed fixtures).
