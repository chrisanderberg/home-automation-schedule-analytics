# Milestone 16 - CTMC estimation + stationary distribution + numerical safeguards

## Canonical scope
This file is canonical for Milestone 16 execution details.

## Objective
- Implement CTMC rate estimation and stationary distribution inference from smoothed statistics.

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
- [ ] Known small-system fixtures validate CTMC and stationary output.
- [ ] Zero/near-zero holding-time regimes are handled with explicit safeguards/tests.
- [ ] Implementation step complete.
- [ ] Analytics tests pass.
- [ ] No unapproved TODO sentinels in production code.

## Non-goals
- No Dagster report orchestration or artifact packaging changes in this milestone.

## Notes
- Include reviewer-oriented comments for numerical invariants and convergence assumptions.
