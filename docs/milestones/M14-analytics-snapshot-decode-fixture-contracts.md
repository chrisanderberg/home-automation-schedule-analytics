# Milestone 14 - Analytics snapshot decode + fixture contracts

## Canonical scope
This file is canonical for Milestone 14 execution details.

## Objective
- Define analytics fixture snapshot contracts and implement deterministic decode scaffolding/tests.

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
- [ ] Fixture snapshots and decode expectations are explicit and deterministic.
- [ ] Error-path tests cover missing tables/schema mismatch/corrupt blobs.
- [ ] Implementation step complete.
- [ ] Analytics tests pass.
- [ ] No unapproved TODO sentinels in production code.

## Non-goals
- [ ] No KDE or CTMC algorithm implementation in this milestone.

## Notes
- Keep decode APIs small and stable; later milestones should build on this contract without changing blob semantics.
