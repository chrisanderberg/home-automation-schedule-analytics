# PLAN.md

## Canonical scope
This file is canonical for milestone ordering and milestone status.

## Milestones

| # | Milestone | Spec file | Status |
|---|---|---|---|
| 0 | Repo bootstrap + unit test harness | `docs/milestones/M00-repo-bootstrap-unit-test-harness.md` | TBD |
| 1 | Dense blob accessors + invariants tests | `docs/milestones/M01-dense-blob-accessors-invariants-tests.md` | TBD |
| 2 | SQLite schema + persistence for controls and aggregates | `docs/milestones/M02-sqlite-schema-persistence-controls-aggregates.md` | TBD |
| 3 | UTC and Local clock bucketing + interval splitting | `docs/milestones/M03-utc-local-clock-bucketing-interval-splitting.md` | TBD |
| 4 | Quarter splitting (UTC calendar quarters) | `docs/milestones/M04-quarter-splitting-utc-calendar-quarters.md` | TBD |
| 5 | Holding ingestion end-to-end (UTC + Local) | `docs/milestones/M05-holding-ingestion-end-to-end-utc-local.md` | TBD |
| 6 | Transition ingestion end-to-end (UTC + Local) | `docs/milestones/M06-transition-ingestion-end-to-end-utc-local.md` | TBD |
| 7 | Solar clocks (mean solar, apparent solar, unequal hours) | `docs/milestones/M07-solar-clocks-mean-apparent-unequal-hours.md` | TBD |
| 8 | Snapshot export for Dagster (daily input artifact) | `docs/milestones/M08-snapshot-export-dagster-daily-artifact.md` | TBD |
| 9 | Dagster project scaffold + daily run | `docs/milestones/M09-dagster-project-scaffold-daily-run.md` | TBD |
| 10 | REST API scaffold + ingestion endpoints | `docs/milestones/M10-rest-api-scaffold-ingestion-endpoints.md` | TBD |
| 11 | Storage path migration + Dagster path alignment (`data/data.sqlite` and `data/snapshots`) | `docs/milestones/M11-storage-path-migration-dagster-path-alignment.md` | TBD |
| 12 | Dual-port API topology + isolated testing API (`:8080` main, `:8081` testing) | `docs/milestones/M12-dual-port-api-topology-isolated-testing-api.md` | TBD |
| 13 | Testing data contract (`testName`/`snapshotName` slug validation, per-test DB/snapshot naming, testing-only reset) | `docs/milestones/M13-testing-data-contract-slug-validation-reset.md` | TBD |

## Execution rule
- Implement each milestone with the two-step process defined in `AGENTS.md`.
