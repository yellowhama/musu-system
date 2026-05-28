# Next Steps: musu-nurikun

> See `C:\Users\empty\MUSU_MCP_AUDIT_2026-05-28.md` for the 2026-05-28 real-usage audit that produced the MCP-related items below.

## P1
- improve mailbox/OAuth bootstrap UX
- turn the sample configs into a more guided setup path or wizard
- declare MCP tool parameter schemas in `cmd/mcp.go` for all 8 tools (currently `WithDescription` only → empty JSON schema → MCP clients cannot pass `list_id`/`email`/`token`/etc.); handler-level validation is already correct, the schema layer just needs to mirror it via `WithString`/`WithNumber`/`Required`

## P2
- add a real mailbox/bootstrap smoke path or staged operator checklist around Gmail/IMAP credentials
- deepen marketer persona integration for campaign tone consistency
- `db.NewStore` should `os.MkdirAll(filepath.Dir(path), 0755)` before `sql.Open` so cwd-isolated MCP invocations (where `projects/<project>/data/` may not exist yet) succeed instead of returning SQLITE_CANTOPEN
- add `json:"…"` tags to `preflight.DoctorResult` so the MCP envelope is snake_case-consistent with the inner Report

## P3
- add Gmail token/bootstrap operator docs and a tighter first-run checklist around the sample configs
- extract shared module(s) for `AgentClient` + `preflight/doctor` + env-loader to remove triple-duplicated logic across the three repos
- run a live mailbox roundtrip (real IMAP/Gmail + Ollama) to close the only remaining "untested in production" gate
- document the `claude mcp add -s user … --env NURIKUN_GMAIL_TOKEN=… --env NURIKUN_UNSUB_SECRET=…` registration pattern in README/AGENTS so the MCP server inherits operator secrets instead of running with default-only config

## Verified Integration Harness
- set `MUSU_NURIKUN_INTEGRATION_AI_URL`
- optionally set `MUSU_NURIKUN_INTEGRATION_MODEL` (verified locally with `llama3.2:1b`)
- run `go test -tags integration ./cmd`
- or run `powershell -ExecutionPolicy Bypass -File .\scripts\run-real-integration.ps1`
