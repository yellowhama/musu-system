# Qualitative Report: musu-nurikun

## Grade
`A`

## Why It Improved
- the stale binary/source mismatch was eliminated by rebuilding the tracked executable
- setup is less fragile because `init` and `doctor --fix` now share the same preset semantics
- machine-readable output makes mailbox/bootstrap failures much easier for agents to diagnose
- `doctor` now fails on missing public delivery settings instead of silently treating them as optional
- local triage/reply smoke coverage now proves the grounded response path can run without a real mailbox
- fake-mailbox command-level smoke coverage now proves both `watch` and compliant `campaign` delivery loops end to end
- a gated real-endpoint integration harness now exists for the `watch` command surface
- the real-integration runner now auto-diagnoses missing local Ollama/OpenAI-compatible runtime candidates instead of failing silently
- the runner now emits machine-readable JSON diagnostics (`-Json -ProbeOnly`) for CI or agent handoff
- the JSON diagnostics now carry stable `issue_codes` so automation can distinguish bind-address misconfiguration from missing installs or timeouts
- the real integration path is now model-configurable through `MUSU_NURIKUN_INTEGRATION_MODEL`
- a real Ollama-backed `watch` integration pass was verified with `llama3.2:1b`
- project-local `.env` loading and preset bootstrap artifacts now reduce mailbox/OAuth setup friction materially
- dead `firstNonEmpty` helper removed and telemetry I/O failures no longer swallowed
- the compiled binary is no longer tracked in git, ending stale-exe drift
- live HTTP roundtrip verified — `serve` + HMAC-signed `/unsubscribe` + web `/confirm` + tamper rejection all e2e-verified, including external openssl-signed payloads interoperating with `compliance.SignUnsub`
- triple-duplicated LLM/config/preflight scaffolding consolidated into `github.com/yellowhama/musu-core@v0.1.0` (env, agent, preflight); internal/agent/config/preflight are now thin wrappers
- `gmail-token` bootstrap closes the manual OAuth provisioning friction; verified live against a real Gmail account (`gmail.users.getProfile` succeeded)
- MCP layer exposes 8 safe ops to other LLM agents; declared parameter schemas make those tools actually callable (previous empty-schema state was effectively a black hole for clients)
- delivery ops (`watch`/`campaign`) are intentionally not on the MCP surface — keeps the opt-in posture defensible at the agent-to-agent integration layer
- Docker deploy bundle brings the full ecosystem up under one compose with ollama, healthchecks, and end-to-end probe verification
- production hardening track shipped at the operator-local layer: x-logging anchor (10MB×3 rotation per service), opt-in `tls` profile (Caddy auto-HTTPS, live-verified through `/healthz` with self-signed cert), opt-in `scheduler` profile (ofelia firing `nurikun watch` on a cron, docker.sock RW fix verified by 2 live firings + clean exit codes), `docker-compose.production.yml` GHCR overlay, `.github/workflows/docker-publish.yml` for multi-arch (amd64+arm64) tag-triggered image publish

## Strong Points
- clear post-pivot product boundary
- strong compliance-first posture
- pluggable mailbox and knowledge-source contracts
- practical recovery path for missing scaffolds

## Concerns
- real mailbox/OAuth bootstrap is still operator-driven even though scaffold generation is better
- `cmd/doctor.go` is growing into a broad orchestrator
- mailbox bootstrap still depends on operator-supplied secrets and URLs that cannot be safely defaulted

## Thermo Verdict
`PASS` (no [CRITICAL]/[HIGH]/[MEDIUM]/[LOW] open, as of 2026-05-28 audit — see `C:\Users\empty\MUSU_THERMONUCLEAR_REVIEW_2026-05-28.md`)

## Immediate Priorities
1. live `watch`/`campaign` roundtrip against real Gmail (Ollama + creds in place; one operator session away)
2. lightweight setup wizard for first-time mailbox + knowledge source picking (the `--mailbox-provider`/`--knowledge-source` presets exist, but a guided TUI/REPL would help newcomers)
3. document the `claude mcp add --env NURIKUN_GMAIL_TOKEN=… --env NURIKUN_UNSUB_SECRET=…` registration pattern in README/AGENTS so MCP clients inherit operator secrets
