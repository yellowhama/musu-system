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

## Docs
- `README.md`: operator quick start
- `SPEC.md`: product contract
- `AGENTS.md`: LLM-oriented usage guidance
- `HANDOFF.md`: implementation handoff
- `CODE_MAP.md`: code/doc index
- `QUALITATIVE_REPORT.md`: current quality verdict
- `NEXT_STEPS.md`: planned follow-up work
