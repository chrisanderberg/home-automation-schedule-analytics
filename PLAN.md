# PLAN.md

## Canonical scope
This file is canonical for milestone ordering and milestone status.

## Milestones

| # | Milestone | Spec file | Status |
|---|---|---|---|
| 0 | Repo bootstrap + unit test harness | `docs/milestones/M00-repo-bootstrap-unit-test-harness.md` | Complete |
| 1 | Dense blob accessors + invariants tests | `docs/milestones/M01-dense-blob-accessors-invariants-tests.md` | Complete |
| 2 | SQLite schema + persistence for controls and aggregates | `docs/milestones/M02-sqlite-schema-persistence-controls-aggregates.md` | Complete |
| 3 | UTC and Local clock bucketing + interval splitting | `docs/milestones/M03-utc-local-clock-bucketing-interval-splitting.md` | Complete |
| 4 | Quarter splitting (UTC calendar quarters) | `docs/milestones/M04-quarter-splitting-utc-calendar-quarters.md` | Complete |
| 5 | Holding ingestion end-to-end (UTC + Local) | `docs/milestones/M05-holding-ingestion-end-to-end-utc-local.md` | Complete |
| 6 | Transition ingestion end-to-end (UTC + Local) | `docs/milestones/M06-transition-ingestion-end-to-end-utc-local.md` | Complete |
| 7 | Solar clocks (mean solar, apparent solar, unequal hours) | `docs/milestones/M07-solar-clocks-mean-apparent-unequal-hours.md` | Complete |
| 8 | Snapshot export for Dagster (daily input artifact) | `docs/milestones/M08-snapshot-export-dagster-daily-artifact.md` | Complete |
| 9 | Dagster project scaffold + daily run | `docs/milestones/M09-dagster-project-scaffold-daily-run.md` | Complete |
| 10 | REST API scaffold + ingestion endpoints | `docs/milestones/M10-rest-api-scaffold-ingestion-endpoints.md` | Complete |
| 11 | Storage path migration + Dagster path alignment (`data/data.sqlite` and `data/snapshots`) | `docs/milestones/M11-storage-path-migration-dagster-path-alignment.md` | Complete |
| 12 | Dual-port API topology + isolated testing API (`:8080` main, `:8081` testing) | `docs/milestones/M12-dual-port-api-topology-isolated-testing-api.md` | Complete |
| 13 | Testing data contract (`testName`/`snapshotName` slug validation, per-test DB/snapshot naming, testing-only reset) | `docs/milestones/M13-testing-data-contract-slug-validation-reset.md` | Complete |
| 14 | Analytics snapshot decode + fixture contracts | `docs/milestones/M14-analytics-snapshot-decode-fixture-contracts.md` | Next |
| 15 | KDE smoothing implementation + golden tests | `docs/milestones/M15-kde-smoothing-golden-tests.md` | Planned |
| 16 | CTMC estimation + stationary distribution + numerical safeguards | `docs/milestones/M16-ctmc-stationary-distribution-safeguards.md` | Planned |
| 17 | Dagster analytics assets/artifacts + idempotent orchestration | `docs/milestones/M17-dagster-analytics-assets-idempotent-orchestration.md` | Planned |
| 18 | Analytics end-to-end validation + report contract hardening | `docs/milestones/M18-analytics-e2e-validation-report-contract.md` | Planned |

## Execution rule
- Implement each milestone with the two-step process defined in `AGENTS.md`.
