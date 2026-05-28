# Next Steps: musu-nurikun

> 2026-05-28 audit items (MCP schemas, db MkdirAll, DoctorResult snake_case envelope, shared module extraction) are now CLOSED. See `MUSU_MCP_AUDIT_2026-05-28.md` and `MUSU_THERMONUCLEAR_REVIEW_2026-05-28.md` for the closed audit chain.

## P1
- live `watch`/`campaign` roundtrip against real Gmail (Ollama + creds in place — one operator session away)
- lightweight setup wizard / guided TUI for first-time mailbox + knowledge source picking

## P2
- deepen marketer persona integration for campaign tone consistency
- a real mailbox/bootstrap smoke path or staged operator checklist around Gmail/IMAP credentials
- document the `claude mcp add -s user … --env NURIKUN_GMAIL_TOKEN=… --env NURIKUN_UNSUB_SECRET=…` registration pattern in README/AGENTS so the MCP server inherits operator secrets instead of running with default-only config

## P3
- a confirm/unsubscribe landing-page UI (currently plain-text responses)
- production hardening on the docker-compose bundle: TLS termination via reverse proxy, log rotation, image registry push, scheduled `watch`/`campaign` (ofelia sidecar or host cron)

## Verified Integration Harness
- set `MUSU_NURIKUN_INTEGRATION_AI_URL`
- optionally set `MUSU_NURIKUN_INTEGRATION_MODEL` (verified locally with `llama3.2:1b`)
- run `go test -tags integration ./cmd`
- or run `powershell -ExecutionPolicy Bypass -File .\scripts\run-real-integration.ps1`
