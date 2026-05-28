# Project Handoff: musu-crawl-ai

## What This Repo Is
`musu-crawl-ai` is the Musu ecosystem's knowledge harvester and local wiki generator. It is the upstream source of truth for `musu-marketer` and `musu-nurikun`.

## Current Truth
- binary: `musu-crawl.exe`
- version constant: `v0.8.0`
- default wiki root: `./wiki`
- default AI contract: OpenAI-compatible endpoint at `--ai-url`
- key recovery path: `doctor --fix`

## What Changed In This Round
- added `doctor`
- added `doctor --fix`
- clarified static setup metadata via `--capability-source`
- added ecosystem workflow docs and machine-readable preflight guidance
- `init` now writes `config.toml`, `PROMPT.md`, and `NEXT_STEPS.md` under each project and returns bootstrap metadata in `--json`
- `research` can now point at a deterministic local search harness through `MUSU_SEARCH_BASE_URL`
- real integration now covers `research` as well as `fetch web` + semantic `search`
- index/telemetry I/O errors are now surfaced — `internal/processor/wiki.go` returns wrapped errors on failed `index.json`/`README.md` writes; `internal/agent/client.go` `logTrace` logs mkdir/write failures to stderr instead of swallowing them
- the compiled `musu-crawl.exe` is no longer tracked in git (already in `.gitignore`; the local file is retained)
- `internal/agent/client.go` + `internal/preflight/doctor.go` now thin wrappers over `github.com/yellowhama/musu-core@v0.1.0` (agent + preflight Probe)
- MCP tool parameter schemas declared (`WithString`/`WithNumber`/`WithBoolean`/`Required`/`Enum`) — clients can actually pass args to `fetch`/`search`/`research`
- `handleResearch` guards empty `question` (no more silent no-op); `handleSearch` emits actionable hint on 0 results
- all 5 ignored `json.Unmarshal` sites now log a stderr warning on parse failure (orchestrator, wiki, harvester/youtube, web/server handleIndex + handleAPIGraph) — silent index corruption closed
- `preflight.DoctorResult` JSON envelope uses snake_case tags (consistent with inner Report)
- Dockerfile added (alpine runtime, digest-pinned base) + brings up under top-level docker-compose alongside ollama/marketer/nurikun. End-to-end `compose up` verified healthy.
- `.github/workflows/docker-publish.yml` added — tag-triggered multi-arch (linux/amd64+arm64) build & push to `ghcr.io/yellowhama/musu-crawl-ai:{tag,latest}` via Buildx + setup-qemu, strict semver tag pattern (`v[0-9]+.[0-9]+.[0-9]+[-*]`)
- production hardening landed at the operator-local layer (top-level `docker-compose.yml` x-logging anchor + opt-in `tls`/`scheduler` profiles, `docker-compose.production.yml` GHCR overlay, `caddy/Caddyfile`, `ofelia/config.ini`) — all live-verified (Caddy TLS reverse-proxy + ofelia firing + ofelia healthcheck)

## Operator Flow
1. `musu-crawl init --out ./wiki --project <name>`
2. `musu-crawl doctor --out ./wiki --project <name>`
3. `musu-crawl fetch <source> <id> --project <name>`
4. `musu-crawl index --out ./wiki --project <name>`
5. `musu-crawl research "<question>" --project <name>`

## Known Constraints
- `--capability-source` is static metadata, not a live reachability probe
- model compatibility is checked by the integration runner, not by ordinary `doctor`
- very large bulk fetches still pay repeated index/vector rewrite cost

## Key Files
- `cmd/root.go`: global flags and JSON mode
- `cmd/init.go`: wiki/project scaffold bootstrap
- `cmd/doctor.go`: preflight and capability metadata output
- `internal/harvester/*`: source-specific fetchers
- `internal/processor/*`: wiki/index/vector processing
- `internal/agent/*`: orchestration and research logic
- `wiki/projects/<project>/config.toml`
- `wiki/projects/<project>/PROMPT.md`
- `wiki/projects/<project>/NEXT_STEPS.md`
