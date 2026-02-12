# PLAN.md

## Plan (milestones and ordering)

Milestones are implemented using the two-step rule defined in `AGENTS.md`.

1. Milestone 0 — Repo bootstrap + unit test harness
2. Milestone 1 — Dense blob accessors + invariants tests
3. Milestone 2 — SQLite schema + persistence for controls and aggregates
4. Milestone 3 — UTC and Local clock bucketing + interval splitting
5. Milestone 4 — Quarter splitting (UTC calendar quarters)
6. Milestone 5 — Holding ingestion end-to-end (UTC + Local)
7. Milestone 6 — Transition ingestion end-to-end (UTC + Local)
8. Milestone 7 — Solar clocks (mean solar, apparent solar, unequal hours)
9. Milestone 8 — Snapshot export for Dagster (daily input artifact)
10. Milestone 9 — Dagster project scaffold + daily run
11. Milestone 10 — REST API scaffold + ingestion endpoints
12. Milestone 11 — Storage path migration + Dagster path alignment (`data/data.sqlite` and `data/snapshots`)
13. Milestone 12 — Dual-port API topology + isolated testing API (`:8080` main, `:8081` testing)
14. Milestone 13 — Testing data contract (`testName`/`snapshotName` slug validation, per-test DB/snapshot naming, testing-only reset)
