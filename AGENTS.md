# AGENTS.md

## Instructions (read first)

### Authority and scope
- This file is the single source of truth for how to work in this repo.
- Product/system requirements are canonical in:
  - `REQUIREMENTS.md` (umbrella + shared guardrails)
  - `AGGREGATION_REQUIREMENTS.md` (aggregation domain)
  - `ANALYTICS_REQUIREMENTS.md` (analytics domain)
- Milestone sequencing is canonical in `PLAN.md`.
- Assumptions and decisions are canonical in `DECISIONS.md`.
- Do not introduce new semantics that contradict the applicable canonical requirements document(s).
- Keep changes tightly scoped to the current milestone in `PLAN.md`.

### Requirement change control
- Requirements may be changed only with explicit user approval.
- It is okay to ask for requirement changes, but do not change requirements without asking first.

### Milestone execution rule (two-step)
Each milestone must be implemented in two steps:

1) Scaffolding + tests
- Create file structure and stubs.
- Add tests (and/or golden fixtures) that define correct behavior.
- Ensure `go test ./...` runs and tests compile (tests may fail if stubs are TODO).

2) Implementation
- Fill in TODOs until tests pass.
- Ensure `go test ./...` passes before completing the milestone.

### Milestone Definition of Done (DoD)
Before marking a milestone complete:
- `go test ./...` passes.
- No placeholder TODO sentinels remain in production code unless explicitly approved by user.
- Non-obvious logic has reviewer-oriented comments explaining intent/invariants.
- Code structure is reviewable (large functions split into focused helpers when practical).

### Assumptions and TBD handling
- If something is not specified, do not guess silently.
- Prefer parameterization when possible.
- Record any necessary assumptions in `DECISIONS.md`.

### Output format expectations (for coding agents)
For any milestone implementation, include:
- files added/changed
- tests added/changed
- commands to run to verify
- assumptions made (if any)
- readability improvements made (if any)
- invariants documented or clarified (if any)

---

## Commands (copy/paste)

These commands must work once the repo is bootstrapped (run from `/aggregation`):

```bash
go test ./...
```

Additional run/build commands will be added once the repo layout is established.

---

## Canonical documents

- Requirements umbrella: `REQUIREMENTS.md`
- Aggregation requirements: `AGGREGATION_REQUIREMENTS.md`
- Analytics requirements: `ANALYTICS_REQUIREMENTS.md`
- Plan: `PLAN.md`
- Assumptions / decisions log: `DECISIONS.md`
- Current execution context: `CURRENT.md`
- Durable progress snapshots: `STATUS.md`
- Testing commands and conventions: `TESTING.md`
- Architecture map: `ARCHITECTURE.md`
- API contracts: `API_CONTRACTS.md`
- Cross-cutting invariants: `INVARIANTS.md`
- Domain glossary: `GLOSSARY.md`

## Skills
A skill is a set of local instructions to follow that is stored in a `SKILL.md` file. Below is the list of skills that can be used. Each entry includes a name, description, and file path so you can open the source for full instructions when using a specific skill.
### Available skills
- skill-creator: Guide for creating effective skills. This skill should be used when users want to create a new skill (or update an existing skill) that extends Codex's capabilities with specialized knowledge, workflows, or tool integrations. (file: /Users/christopheranderberg/.codex/skills/.system/skill-creator/SKILL.md)
- skill-installer: Install Codex skills into $CODEX_HOME/skills from a curated list or a GitHub repo path. Use when a user asks to list installable skills, install a curated skill, or install a skill from another repo (including private repos). (file: /Users/christopheranderberg/.codex/skills/.system/skill-installer/SKILL.md)
### How to use skills
- Discovery: The list above is the skills available in this session (name + description + file path). Skill bodies live on disk at the listed paths.
- Trigger rules: If the user names a skill (with `$SkillName` or plain text) OR the task clearly matches a skill's description shown above, you must use that skill for that turn. Multiple mentions mean use them all. Do not carry skills across turns unless re-mentioned.
- Missing/blocked: If a named skill isn't in the list or the path can't be read, say so briefly and continue with the best fallback.
- How to use a skill (progressive disclosure):
  1) After deciding to use a skill, open its `SKILL.md`. Read only enough to follow the workflow.
  2) When `SKILL.md` references relative paths (e.g., `scripts/foo.py`), resolve them relative to the skill directory listed above first, and only consider other paths if needed.
  3) If `SKILL.md` points to extra folders such as `references/`, load only the specific files needed for the request; don't bulk-load everything.
  4) If `scripts/` exist, prefer running or patching them instead of retyping large code blocks.
  5) If `assets/` or templates exist, reuse them instead of recreating from scratch.
- Coordination and sequencing:
  - If multiple skills apply, choose the minimal set that covers the request and state the order you'll use them.
  - Announce which skill(s) you're using and why (one short line). If you skip an obvious skill, say why.
- Context hygiene:
  - Keep context small: summarize long sections instead of pasting them; only load extra files when needed.
  - Avoid deep reference-chasing: prefer opening only files directly linked from `SKILL.md` unless you're blocked.
  - When variants exist (frameworks, providers, domains), pick only the relevant reference file(s) and note that choice.
- Safety and fallback: If a skill can't be applied cleanly (missing files, unclear instructions), state the issue, pick the next-best approach, and continue.
