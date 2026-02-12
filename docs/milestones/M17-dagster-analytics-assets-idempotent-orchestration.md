# Milestone 17 - Dagster analytics assets/artifacts + idempotent orchestration

## Canonical scope
This file is canonical for Milestone 17 execution details.

## Objective
- Build Dagster assets/materializations for analytics outputs with idempotent trigger handling.

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
- [ ] Asset materializations expose snapshot provenance and output-shape metadata.
- [ ] Sensor and schedule behavior is tested for idempotent duplicate-trigger handling.
- [ ] Implementation step complete.
- [ ] Analytics tests pass.
- [ ] No unapproved TODO sentinels in production code.

## Non-goals
- [ ] No additional inference algorithms beyond KDE + CTMC + stationary distribution.

## Notes
- Prefer stable run keys/tags that encode snapshot identity to avoid duplicate work.
