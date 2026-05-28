# Project Handoff: musu-marketer

## What This Repo Is
`musu-marketer` is the Musu ecosystem's campaign drafting engine. It consumes a grounded wiki, keeps project-local personas and campaign state, and outputs publishable copy through local or webhook publishers.

## Current Truth
- binary: `musu-marketer.exe`
- version constant: `v2.0.1`
- default AI contract: OpenAI-compatible endpoint at `--ai-url`
- default wiki contract: sibling `../musu-crawl-ai/wiki` auto-discovery when `--wiki` is omitted
- key recovery path: `doctor --fix`

## What Changed In This Round
- added `doctor`
- added `--json`
- added `doctor --fix`
- changed topic readiness from filename-only to ranked `index.json` + body-content matching
- added test coverage for indexed/content topic lookup
- `init` now writes a project-local `NEXT_STEPS.md` and returns structured bootstrap metadata in `--json`
- embedded the Marketing Bible so draft execution stays stable outside the repo root
- telemetry I/O errors in `internal/agent/client.go` `logTrace` are now logged to stderr instead of swallowed
- the compiled `musu-marketer.exe` is no longer tracked in git (already in `.gitignore`; the local file is retained)
- `internal/agent/client.go` + `internal/preflight/doctor.go` now thin wrappers over `github.com/yellowhama/musu-core@v0.1.0`
- MCP tool parameter schemas declared for `draft_campaign` + `list_campaigns` — clients can actually pass `topic`/`project`/`persona`
- `handleDraft` guards empty `topic` (was silently proceeding)
- `db.NewStore` MkdirAll(parent) before sql.Open — cwd-isolated MCP invocations no longer fail with SQLITE_CANTOPEN
- `preflight.DoctorResult` JSON envelope uses snake_case tags (consistent with inner Report)
- Dockerfile added (alpine runtime, digest-pinned base) + brings up under top-level docker-compose alongside ollama/crawl/nurikun. End-to-end `compose up` verified healthy.
- `.github/workflows/docker-publish.yml` added — tag-triggered multi-arch (linux/amd64+arm64) build & push to `ghcr.io/yellowhama/musu-marketer:{tag,latest}` via Buildx + setup-qemu, strict semver tag pattern (`v[0-9]+.[0-9]+.[0-9]+[-*]`)
- production hardening landed at the operator-local layer (top-level `docker-compose.yml` x-logging anchor + opt-in `tls`/`scheduler` profiles, `docker-compose.production.yml` GHCR overlay) — all live-verified (Caddy TLS reverse-proxy + ofelia firing + ofelia healthcheck)

## Operator Flow
1. `musu-marketer init --project <name>`
2. `musu-marketer doctor --project <name> --topic "<topic>"`
3. `musu-marketer draft "<topic>" --persona <persona> --project <name>`
4. `musu-marketer publish <id> --platform local|webhook`

## Known Constraints
- topic readiness is now weighted/ranked, but it is still not a full semantic retriever
- `doctor` still mixes reporting and fixing in one command file
- local smoke coverage exists, but topic retrieval still depends on lexical evidence rather than vector retrieval

## Key Files
- `cmd/root.go`: global flags, wiki auto-discovery
- `cmd/init.go`: project bootstrap
- `cmd/doctor.go`: preflight and scaffold repair
- `cmd/output.go`: JSON success/error envelope
- `internal/bridge/wiki.go`: wiki lookup and topic matching
- `internal/agent/*`: strategist/copywriter/critic logic
- `internal/agent/skills.go`: embedded Marketing Bible loader
- `projects/<project>/NEXT_STEPS.md`: generated project-local bootstrap guide
