# Qualitative Report: musu-crawl-ai

## Grade
`A`

## Why It Improved
- the tool is now much easier for agents to preflight and recover locally
- `doctor --json` and `doctor --fix` make setup failures legible
- source setup metadata is now labeled honestly as static capability data
- local fetch CLI smoke now proves a real `web` harvest path against a controllable HTTP source
- URL normalization is safer for `http://` targets on Windows filesystem paths
- a gated real-endpoint integration harness now exists for fetch + semantic search command verification
- the real-integration runner now auto-diagnoses missing local Ollama/OpenAI-compatible runtime candidates instead of failing silently
- the runner now emits machine-readable JSON diagnostics (`-Json -ProbeOnly`) for CI or agent handoff
- the JSON diagnostics now carry stable `issue_codes` so automation can distinguish bind-address misconfiguration from missing installs or timeouts
- the real integration path is now model-configurable through `MUSU_CRAWL_INTEGRATION_EMBED_MODEL`
- the real integration path is now also chat-model-configurable through `MUSU_CRAWL_INTEGRATION_CHAT_MODEL`
- a real Ollama-backed `fetch web` + semantic `search` integration pass was verified with `nomic-embed-text`
- a real Ollama-backed `research` command integration pass was verified with a deterministic local search harness
- a command-flag state leak was caught under integration load and fixed in `search_test.go`
- the integration doctor now treats `model` and `model:latest` as equivalent, avoiding false negatives against Ollama
- index/telemetry I/O errors are no longer silent — failed `index.json`/`README` persistence in `wiki.go` returns an error and telemetry `logTrace` logs failures to stderr
- the compiled binary is no longer tracked in git, ending recurring stale-exe-vs-source drift
- triple-duplicated AgentClient + preflight Probe scaffolding consolidated into `github.com/yellowhama/musu-core@v0.1.0`; internal/agent + internal/preflight are now thin wrappers
- MCP tool surface is finally callable from clients — parameter schemas declared (was empty before, blocking arg-passing)
- `research` no longer silently no-ops on missing question; `search` distinguishes "no matches" from "nothing indexed" with an actionable hint
- 5 ignored `json.Unmarshal` sites across orchestrator/wiki/youtube/web/server now surface parse failures to stderr — silent index corruption sealed
- Docker deploy bundle brings the full ecosystem up under one compose with ollama, healthchecks, and end-to-end probe verification
- production hardening track shipped at the operator-local layer: x-logging anchor (10MB×3 rotation per service), opt-in `tls` profile (Caddy auto-HTTPS via Let's Encrypt or internal CA for local dev), opt-in `scheduler` profile (ofelia sidecar), `docker-compose.production.yml` GHCR overlay, `.github/workflows/docker-publish.yml` for multi-arch (amd64+arm64) tag-triggered image publish

## Strong Points
- broad harvesting surface
- clear wiki-first architecture
- deterministic machine-readable output path
- strong position as the upstream knowledge source for the other two repos
- local command and orchestrator smoke coverage now prove bootstrap, fetch, and search surfaces without external services

## Concerns
- AI endpoint probing is now correctly flag-aware and model-aware, but it is still limited to runtime/model contract checks rather than full source-contract validation
- source capability output is static, not a live probe
- doctor logic is starting to accumulate multiple roles in one command file

## Thermo Verdict
`PASS` (no [CRITICAL]/[HIGH]/[MEDIUM]/[LOW] open, as of 2026-05-28 audit — see `C:\Users\empty\MUSU_THERMONUCLEAR_REVIEW_2026-05-28.md`)

## Immediate Priorities
1. decide whether live source probes should exist at all
2. split doctor reporting helpers from static capability metadata
3. optimize bulk-crawl index/vector rewrite cost (O(N²) write amplification on the live-sync path)
4. (production hardening on the docker-compose bundle: TLS termination, log rotation, image registry push, scheduled batch crawls)
