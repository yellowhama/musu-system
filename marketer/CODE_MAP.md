# Code And Doc Map: musu-marketer

## Runtime Entry Points
- `main.go`: CLI bootstrap
- `cmd/root.go`: global flags and environment initialization
- `cmd/init.go`: project/database/persona bootstrap
- `cmd/doctor.go`: wiki/project/AI/topic preflight
- `cmd/draft.go`: main draft command
- `cmd/autopilot.go`: higher-level orchestration flow
- `cmd/publish.go`: publish adapters
- `cmd/mcp.go`: MCP server (exposes `draft_campaign` + `list_campaigns` with declared parameter schemas)
- `Dockerfile`: alpine runtime image (digest-pinned golang build stage); see top-level `docker-compose.yml`
- `.github/workflows/docker-publish.yml`: tag-triggered multi-arch (amd64+arm64) GHCR publish to `ghcr.io/yellowhama/musu-marketer`

## Core Packages
- `internal/agent`: strategist, copywriter, critic, shared AI client
- `internal/agent/skills.go`: embedded Marketing Bible loader
- `internal/bridge`: wiki integration and topic lookup
- `internal/db`: SQLite persistence for campaigns
- `internal/publisher`: local and webhook publishing
- `internal/api`: HTTP server surface

## Project Data Layout
- `projects/<project>/campaigns`
- `projects/<project>/personas`
- `projects/<project>/data/marketer.db`
- `projects/<project>/published`
- `projects/<project>/NEXT_STEPS.md`

## Docs
- `README.md`: operator quick start
- `SPEC.md`: product contract
- `AGENTS.md`: LLM-oriented usage guidance
- `HANDOFF.md`: implementation handoff
- `CODE_MAP.md`: code/doc index
- `QUALITATIVE_REPORT.md`: current quality verdict
- `NEXT_STEPS.md`: planned follow-up work
- `INTEGRATION.md`: API-facing integration notes

## Examples
- `examples/sample-wiki/*`: tiny grounded wiki fixture for smoke checks
