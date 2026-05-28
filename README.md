# musu — knowledge → voice → hand

Monorepo for the Musu ecosystem.

| Service | Role | Image |
|---|---|---|
| [`crawl-ai/`](./crawl-ai) | Knowledge harvester + local wiki | `ghcr.io/yellowhama/musu-crawl-ai` |
| [`marketer/`](./marketer) | Campaign drafting engine | `ghcr.io/yellowhama/musu-marketer` |
| [`nurikun/`](./nurikun)   | Opt-in email agent (CS inbound + double-opt-in newsletter) | `ghcr.io/yellowhama/musu-nurikun` |
| [`core/`](./core)         | Shared library (env / agent / preflight) | n/a (Go module) |

## Layout

```
musu-system/
├── go.work                # workspace pinning the 4 modules
├── core/                  # github.com/yellowhama/musu-system/core
├── crawl-ai/              # github.com/yellowhama/musu-system/crawl-ai
├── marketer/              # github.com/yellowhama/musu-system/marketer
├── nurikun/               # github.com/yellowhama/musu-system/nurikun
├── deploy/                # operator deploy bundle (docker-compose, caddy, ofelia)
├── docs/
│   ├── audits/            # MCP audit + thermonuclear reviews
│   └── workflows/         # cross-tool skills + runtime guides
└── .github/workflows/     # CI: ci.yml + docker-publish.yml (path-based tag matrix)
```

## Build

```bash
# all modules
for d in core crawl-ai marketer nurikun; do (cd $d && go build ./... && go test ./...) ; done

# or with workspace
go build ./crawl-ai/... ./marketer/... ./nurikun/... ./core/...
```

## Deploy

See [`deploy/README.md`](./deploy/README.md). Quick path (operator host):

```powershell
cd deploy
docker compose up -d --build
# or with hardening profiles:
docker compose --profile tls --profile scheduler up -d --build
```

## Versioning + release

Tag convention: `<service>/vMAJOR.MINOR.PATCH` (e.g. `marketer/v2.0.5`, `core/v0.2.0`).
Tag push triggers `.github/workflows/docker-publish.yml` → multi-arch build →
`ghcr.io/yellowhama/musu-<service>:<version>` + `:latest`. Trivy CRITICAL/HIGH
scan results land in the repo's Security tab.

For nested Go modules: `go get github.com/yellowhama/musu-system/core@core/v0.2.0`.

## History note

Consolidated 2026-05-28 from four prior repos:
- `github.com/yellowhama/musu-core`      → `core/`
- `github.com/yellowhama/musu-crawl-ai`  → `crawl-ai/`
- `github.com/yellowhama/musu-marketer`  → `marketer/`
- `github.com/yellowhama/musu-nurikun`   → `nurikun/`

Original commit SHAs are preserved via `git subtree add` — every legacy
commit is still reachable in the monorepo by its original hash. **However**,
path-filtered `git log -- <prefix>/<file>` will only show the subtree merge
commit, NOT the file's true history. This is because pre-merge commits
recorded the file at its root-relative path (e.g. `cmd/fetch.go`), not the
prefixed path (`crawl-ai/cmd/fetch.go`).

To trace file history correctly, pick one:

```bash
# 1. Origin-relative path on the full graph (recommended)
git log --all --follow -- cmd/fetch.go

# 2. Specific origin SHA -> walk back
git log <merge-commit-sha> -- cmd/fetch.go

# 3. Subtree split the prefix into a temporary branch
git subtree split --prefix=crawl-ai -b crawl-ai-history
git log crawl-ai-history -- cmd/fetch.go
```

The legacy origin repos (kept read-only during the 2-week observation
window) remain a clean reference for plain `git log` UX during that time.
