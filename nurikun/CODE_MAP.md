# Code And Doc Map: musu-nurikun

## Runtime Entry Points
- `main.go`: CLI bootstrap
- `cmd/root.go`: global flags and environment wiring
- `cmd/init.go`: scaffold/config bootstrap
- `cmd/doctor.go`: config, mailbox, knowledge, and AI preflight
- `cmd/watch.go`: inbound mailbox loop
- `cmd/campaign.go`: outbound opt-in sending
- `cmd/lists.go`, `cmd/subscribe.go`, `cmd/confirm.go`, `cmd/suppress.go`: consent and list management
- `cmd/serve.go`: confirm/unsubscribe web endpoints
- `cmd/mcp.go`: MCP server exposing 8 safe ops (delivery ops CLI-only)
- `cmd/gmail_token.go`: one-off OAuth bootstrap (local loopback callback)
- `Dockerfile`: alpine runtime image (digest-pinned golang build stage); see top-level `docker-compose.yml`
- `.github/workflows/docker-publish.yml`: tag-triggered multi-arch (amd64+arm64) GHCR publish to `ghcr.io/yellowhama/musu-nurikun`

## Core Packages
- `internal/config`: config loading
- `internal/mailbox`: IMAP/Gmail providers
- `internal/knowledge`: crawl-ai/folder/none sources
- `internal/policy`: send/reply decision logic
- `internal/compliance`: unsubscribe, labels, rate limiting
- `internal/agent`: responder logic
- `internal/db`: SQLite persistence

## Project Data Layout
- `projects/<project>/config.yaml`
- `projects/<project>/data/nurikun.db`
- `projects/<project>/knowledge`
- `projects/<project>/SETUP.md`
- `projects/<project>/knowledge/README.md` for folder-based setups

## Docs
- `README.md`: operator quick start
- `SPEC.md`: product contract
- `AGENTS.md`: LLM-oriented usage guidance
- `HANDOFF.md`: implementation handoff
- `CODE_MAP.md`: code/doc index
- `QUALITATIVE_REPORT.md`: current quality verdict
- `NEXT_STEPS.md`: planned follow-up work

## Examples
- `examples/config.imap.yaml`: sample IMAP/SMTP project config
- `examples/config.gmail.yaml`: sample Gmail OAuth project config
