# Milestone 18 - Analytics end-to-end validation + report contract hardening

## Canonical scope
This file is canonical for Milestone 18 execution details.

## Objective
- Add end-to-end validation and lock down report/artifact contract behavior.

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
- [ ] End-to-end test proves snapshot input -> analytics outputs with expected metadata.
- [ ] Report/artifact schema contract checks are explicit and versioned.
- [ ] Implementation step complete.
- [ ] Analytics tests pass.
- [ ] No unapproved TODO sentinels in production code.

## Non-goals
- No changes to aggregation ingestion/storage semantics.

## Notes
- Keep contract tests strict enough to catch accidental schema drift.
