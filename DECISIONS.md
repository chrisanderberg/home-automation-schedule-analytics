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
