# Project Handoff: musu-nurikun

## What This Repo Is
`musu-nurikun` is the Musu ecosystem's email agent. It handles inbound support mail and opt-in campaign delivery. It is explicitly post-pivot: no forged identities, no signup botting, no cold outreach.

## Current Truth
- binary: `musu-nurikun.exe`
- version constant: `v0.3.0`
- default AI contract: OpenAI-compatible endpoint at `--ai-url`
- project config: `projects/<project>/config.yaml`
- key recovery path: `doctor --fix`

## What Changed In This Round
- added `doctor`
- added `--json`
- added `doctor --fix`
- aligned `doctor --fix` with `init` by accepting `--mailbox-provider` and `--knowledge-source`
- rebuilt the tracked exe so the binary matches the current source/README command surface
- `init` now writes a project-local `SETUP.md`, optional folder knowledge guide, and machine-readable bootstrap metadata
- `init` now also writes `.env.example`, `bootstrap.ps1`, and Gmail `oauth/README.md`
- runtime config now loads project-local `.env` as a secrets layer above `config.yaml`

## Operator Flow
1. `musu-nurikun init --project <name> --mailbox-provider imap|gmail --knowledge-source crawlai|folder|none`
2. fill mailbox, sender, and public delivery settings in `projects/<name>/config.yaml`
3. `musu-nurikun doctor --project <name>`
4. `musu-nurikun watch` or `musu-nurikun campaign ...`

## Known Constraints
- mailbox/OAuth bootstrap still requires manual operator credentials even though the scaffold is more guided
- `doctor` is comprehensive, but it is still one command file doing report + fix orchestration
- `public_base_url` and `unsub_secret` remain manual to avoid accidental weak defaults, and `doctor` now treats them as blocking readiness requirements

## Key Files
- `cmd/root.go`: global flags and JSON mode
- `cmd/init.go`: project/config bootstrap
- `cmd/doctor.go`: config/mailbox/knowledge/AI preflight
- `cmd/output.go`: JSON success/error envelope
- `internal/config/config.go`: config loading
- `internal/mailbox/*`: IMAP/Gmail integrations
- `internal/knowledge/*`: crawl-ai/folder/none sources
- `internal/compliance/*`: unsubscribe/rate-limit/policy helpers
- `projects/<project>/SETUP.md`
- `projects/<project>/knowledge/README.md` when `knowledge_source=folder`
