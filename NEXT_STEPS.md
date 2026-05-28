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
- ~~production hardening on the docker-compose bundle: TLS termination via reverse proxy, log rotation, image registry push, scheduled `watch`/`campaign` (ofelia sidecar or host cron)~~ — CLOSED 2026-05-28. Caddy `tls` profile (auto-HTTPS) verified end-to-end through `/healthz`, x-logging anchor (10MB×3 rotation), `.github/workflows/docker-publish.yml` for GHCR push, ofelia `scheduler` profile firing `nurikun watch` (live verified at @every 30s, then reverted to @every 10m; docker.sock RW fix proven).
- real-domain Let's Encrypt verification — replace `localhost` block in `caddy/Caddyfile` with a public DNS name + verify Let's Encrypt issues a real cert (current verification used Caddy's internal CA / self-signed)
- first real GHCR push validation — operator pushes a `vX.Y.Z` tag and confirms the workflow publishes `ghcr.io/yellowhama/musu-nurikun:vX.Y.Z` + `:latest`
- enable `[job-exec "nurikun-campaign-weekly-digest"]` in `ofelia/config.ini` only after a real subscriber list exists and a manual `campaign send` dry-run has passed

## Verified Integration Harness
- set `MUSU_NURIKUN_INTEGRATION_AI_URL`
- optionally set `MUSU_NURIKUN_INTEGRATION_MODEL` (verified locally with `llama3.2:1b`)
- run `go test -tags integration ./cmd`
- or run `powershell -ExecutionPolicy Bypass -File .\scripts\run-real-integration.ps1`
