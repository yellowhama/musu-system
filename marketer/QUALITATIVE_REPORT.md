# Qualitative Report: musu-marketer

## Grade
`A`

## Why It Improved
- preflight is now real enough to prevent obvious drafting failures
- wiki path hardcoding was reduced through sibling auto-discovery and env overrides
- topic readiness no longer lies by checking only filenames
- project bootstrap and JSON output make the tool much more agent-friendly
- local draft smoke coverage now proves the main strategist/copywriter/critic path can complete against a grounded fixture
- command-level draft smoke now proves the real Cobra `draft` surface saves campaigns correctly against a fake AI + grounded wiki
- a gated real-endpoint integration harness now exists for the same `draft` command surface
- the real-integration runner now auto-diagnoses missing local Ollama/OpenAI-compatible runtime candidates instead of failing silently
- the runner now emits machine-readable JSON diagnostics (`-Json -ProbeOnly`) for CI or agent handoff
- the JSON diagnostics now carry stable `issue_codes` so automation can distinguish bind-address misconfiguration from missing installs or timeouts
- the real integration path is now model-configurable through `MUSU_MARKETER_INTEGRATION_MODEL`
- a real Ollama-backed CLI `draft` integration pass was verified with `llama3.2:1b`
- topic lookup now ranks title/tag/summary matches ahead of weaker body-only hits
- telemetry `logTrace` I/O errors are no longer swallowed (mkdir/write failures logged to stderr)
- the compiled binary is no longer tracked in git, ending stale-exe drift
- triple-duplicated AgentClient + preflight Probe scaffolding consolidated into `github.com/yellowhama/musu-core@v0.1.0`; internal/agent + internal/preflight are now thin wrappers
- MCP tool surface is callable from clients — parameter schemas declared (was empty, blocking arg-passing); `draft` guards empty topic
- `db.NewStore` MkdirAll(parent) closes the SQLITE_CANTOPEN failure mode for cwd-isolated invocations
- Docker deploy bundle brings the full ecosystem up under one compose with ollama, healthchecks, and end-to-end probe verification
- production hardening track shipped at the operator-local layer: x-logging anchor (10MB×3 rotation per service), opt-in `tls` profile (Caddy auto-HTTPS), opt-in `scheduler` profile (ofelia sidecar), `docker-compose.production.yml` GHCR overlay, `.github/workflows/docker-publish.yml` for multi-arch (amd64+arm64) tag-triggered image publish

## Strong Points
- clear project siloing
- grounded draft workflow tied to `musu-crawl-ai`
- deterministic machine-readable output path
- publish surface stays small and understandable

## Concerns
- topic readiness is now ranked lexical retrieval, but it is still not semantic/vector retrieval
- `cmd/doctor.go` is beginning to accrete too many responsibilities
- publish adapters are still shallow beyond local/webhook

## Thermo Verdict
`PASS` (no [CRITICAL]/[HIGH]/[MEDIUM]/[LOW] open, as of 2026-05-28 audit — see `C:\Users\empty\MUSU_THERMONUCLEAR_REVIEW_2026-05-28.md`)

## Immediate Priorities
1. decide whether topic retrieval should stay lexical or graduate to semantic/vector retrieval
2. grow the wiki fixture into a richer multi-topic sample corpus
3. improve publish adapters beyond local/webhook
4. (production hardening on the docker-compose bundle: TLS termination, log rotation, image registry push)
