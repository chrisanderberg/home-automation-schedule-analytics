# DECISIONS.md

## Assumptions / Decisions log

When new assumptions are made during implementation, append entries here:

- Date:
- Milestone:
- Assumption:
- Why needed:
- Impact/risk:
- Resolution (TBD / decided / update REQUIREMENTS.md):

- Date: 2026-02-10
- Milestone: 0
- Assumption: Go module path set to `home-automation-analytics/aggregation`.
- Why needed: `go.mod` requires a module path to initialize the Go project.
- Impact/risk: May need to change if a canonical VCS import path is desired.
- Resolution (TBD / decided / update REQUIREMENTS.md): TBD

- Date: 2026-02-10
- Milestone: 2
- Assumption: `controls.state_labels` stored as JSON-encoded string when present.
- Why needed: Requirements define optional labels but not an encoding.
- Impact/risk: Changing encoding later requires migration.
- Resolution (TBD / decided / update REQUIREMENTS.md): TBD

- Date: 2026-02-12
- Milestone: Documentation refactor
- Assumption: Requirements are split into domain-specific files while keeping an umbrella index.
- Why needed: Aggregation and analytics scope matured asymmetrically; a single mixed requirements file reduced clarity.
- Impact/risk: Existing references to `REQUIREMENTS.md` may become stale if not updated.
- Resolution (TBD / decided / update REQUIREMENTS.md): decided

- Date: 2026-02-12
- Milestone: Planning baseline update
- Assumption: Milestones 0-13 are marked complete based on implemented code/tests present in repo.
- Why needed: `PLAN.md` had all milestones as `TBD` despite completed implementation footprint.
- Impact/risk: If historical milestone evidence is incomplete, statuses may need revision.
- Resolution (TBD / decided / update REQUIREMENTS.md): TBD
