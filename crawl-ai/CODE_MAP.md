# Code And Doc Map: musu-crawl-ai

## Runtime Entry Points
- `main.go`: CLI bootstrap
- `cmd/root.go`: global flags and JSON/error behavior
- `cmd/init.go`: local wiki scaffold bootstrap
- `cmd/doctor.go`: preflight and static capability metadata
- `cmd/fetch.go`: source harvesting
- `cmd/index.go`: index/vector refresh
- `cmd/research.go`: orchestrated research loop
- `cmd/serve.go`: local web dashboard
- `cmd/mcp.go`: MCP server (tool defs in `internal/agent/mcp_server.go`)
- `Dockerfile`: alpine runtime image (digest-pinned golang build stage); see top-level `docker-compose.yml`
- `.github/workflows/docker-publish.yml`: tag-triggered multi-arch (amd64+arm64) GHCR publish to `ghcr.io/yellowhama/musu-crawl-ai`

## Core Packages
- `internal/harvester`: source fetchers
- `internal/processor`: wiki/index/vector processing
- `internal/agent`: planner/searcher/fetcher/compiler/orchestrator
- `internal/utils`: config, logger, HTTP, OCR, summarization helpers
- `internal/web`: local dashboard server

## Data Layout
- `wiki/index.json`
- `wiki/musu.bleve`
- `wiki/projects/<project>/...`
- `wiki/projects/<project>/config.toml`
- `wiki/projects/<project>/PROMPT.md`
- `wiki/projects/<project>/NEXT_STEPS.md`

## Docs
- `README.md`: operator quick start
- `SPEC.md`: product contract
- `AGENTS.md`: LLM-oriented usage guidance
- `HANDOFF.md`: implementation handoff
- `CODE_MAP.md`: code/doc index
- `QUALITATIVE_REPORT.md`: current quality verdict
- `NEXT_STEPS.md`: planned follow-up work
- `ECOSYSTEM_WORKFLOW.md`: cross-tool usage flow
- `THERMONUCLEAR_REVIEW.md`: local harsh-review rubric
