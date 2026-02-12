# Aggregation Service Reviewer Guide

This package implements ingestion and aggregation for home automation control data.

## Canonical references
- Architecture and data flow:
  - `../ARCHITECTURE.md`
- Invariants and index math:
  - `../INVARIANTS.md`
- HTTP contracts:
  - `../API_CONTRACTS.md`
- Test commands and data conventions:
  - `../TESTING.md`
- Requirements:
  - `../REQUIREMENTS.md`

## Where to review behavior in code
- Dense blob index layout: `blob/accessor.go`
- UTC/local/solar bucket mapping and interval splitting: `bucketing`
- Quarter splitting logic: `quarter/quarter.go`
- Storage transactions and upsert semantics: `storage/store.go`
- Ingestion orchestration: `ingest`
- API request contracts and strict JSON handling: `api`
- Snapshot copy and naming behavior: `snapshot/snapshot.go`
