# musu-marketer Spec (STATUS: v2.0.1 PRELIGHTENED)

## Goal
`musu-marketer` is the "Voice" of the Musu ecosystem. It turns grounded wiki knowledge from `musu-crawl-ai` into campaign drafts, persona-shaped copy, and publishable marketing assets.

## Current Product Truth

### Drafting Surface
- `draft [topic]` creates a strategic brief and marketing copy from the current project context.
- `autopilot [topic]` is a higher-level orchestration path that assumes research material already exists or can be prepared in the wider Musu workflow.
- `persona` manages local tone/identity files under `projects/<project>/personas`.
- the Marketing Bible is embedded in the binary, so draft execution does not depend on the shell's working directory.

### Preflight / Recovery
- `init` creates the local project scaffold, default persona, config, and `NEXT_STEPS.md`.
- `doctor` verifies the wiki path, project directory, SQLite DB, AI endpoint, and optional topic readiness.
- `doctor --fix` can safely create the local project scaffold and database.
- `doctor --topic "..."` now uses `wiki/index.json` plus markdown body content, not just filenames.
- `--json` produces deterministic output for agents and automation.
- `init --json` returns scaffold paths and recommended next commands.

### Wiki Contract
- `--wiki` can be omitted.
- When omitted, marketer auto-discovers a sibling `../musu-crawl-ai/wiki`, then falls back to environment overrides and local candidates.
- Topic readiness searches titles, summaries, tags, paths, and body content.

## Completed Milestones
- [x] Project-scoped bootstrap with persona/database scaffold
- [x] OpenAI-compatible AI backend configuration
- [x] Publish adapters (`local`, `webhook`)
- [x] JSON output mode
- [x] Doctor command with `--fix`
- [x] Topic readiness backed by index/body content
- [x] Wiki auto-discovery for sibling `musu-crawl-ai`
- [x] Local draft smoke coverage against a grounded wiki fixture
- [x] Telemetry `logTrace` I/O errors are now surfaced to stderr (no silently-lost traces)
- [x] Compiled `musu-marketer.exe` binary is no longer tracked in git
- [x] **Shared module integration**: `internal/agent/client.go`, `internal/preflight/doctor.go` now thin wrappers over `github.com/yellowhama/musu-core@v0.1.0`.
- [x] **MCP tool parameter schemas declared** — `draft_campaign` / `list_campaigns` now expose `WithString`/`Required` so MCP clients can pass `topic`/`project`/`persona`.
- [x] **`handleDraft` empty-input guard** — empty `topic` rejected with `"topic is required"` (no more silent slide into the LLM pipeline).
- [x] **`db.NewStore` MkdirAll(parent)** — cwd-isolated invocations (MCP servers) no longer fail with SQLITE_CANTOPEN on missing project dirs.
- [x] **`preflight.DoctorResult` JSON envelope** — snake_case `json` tags for consistency with the inner Report.
- [x] **Docker deploy bundle** — Dockerfile (alpine, digest-pinned golang) brings up under top-level docker-compose with ollama/crawl/nurikun. End-to-end `compose up` verified healthy.

## Known Constraints
- Topic readiness is still heuristic substring matching, not ranked retrieval.
- `doctor` is useful, but it is not a substitute for real content quality review.

## Next Work
1. Replace substring-based topic readiness with ranked or indexed retrieval.
2. Split `doctor` into report/fix helpers before it grows into a command blob.
3. Extend the smoke fixture into a richer multi-topic sample corpus.

---
**Build Date:** 2026-05-27
**Status:** 🗣️ VOICE READY
