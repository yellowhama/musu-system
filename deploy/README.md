# Musu Ecosystem — Docker Deploy Guide

> One-command bring-up of Ollama + crawl-ai + marketer + nurikun.
> Targets Docker Desktop on Windows (WSL2 + NVIDIA Container Toolkit) but the
> compose file is portable to Linux hosts with the NVIDIA runtime.

## What this brings up

| Service | Image / Build | Port | Purpose |
|---|---|---|---|
| `ollama` | `ollama/ollama:latest` (GPU) | 11434 | OpenAI-compatible chat/embed inference. Models persisted in `ollama_data` volume. |
| `crawl` | `./musu-crawl-ai` Dockerfile | **8090** (→ 8080 in container) | `musu-crawl serve` — wiki dashboard + bleve search backend. Host port 8080 is commonly held on Windows by Docker Desktop's own backend, so we publish at 8090. |
| `marketer` | `./musu-marketer` Dockerfile | 8081 | `musu-marketer serve` — REST API for campaign drafting. |
| `nurikun` | `./musu-nurikun` Dockerfile | 8088 | `musu-nurikun serve` — HMAC-signed `/unsubscribe` + `/confirm` web endpoints. |

All three musu services point at `http://ollama:11434/v1` (in-network DNS). `marketer` and `nurikun` mount `./wiki` (from `crawl`) and `./oauth` (Gmail creds) as needed.

> ⚠️ **`crawl serve` is the DASHBOARD ONLY.** It does not auto-harvest. To
> actually ingest content you must invoke `docker compose run --rm crawl
> fetch <source> <url> --project <name>` separately (or `research "..."`).
> The dashboard simply surfaces whatever the local wiki already contains.

> ⚠️ **`nurikun watch` and `nurikun campaign` are NOT daemons.** The compose
> file runs `nurikun serve` (HMAC unsubscribe / confirm web endpoints) as the
> always-on container. Inbound triage and outbound delivery are one-shot:
> `docker compose run --rm nurikun watch --project default --limit 10` and
> `docker compose run --rm nurikun campaign send --list 1 --name <...>`. You
> must schedule those yourself (Windows Task Scheduler, cron, or an ofelia
> sidecar). This is intentional per nurikun's opt-in posture — a human or
> a deliberate scheduler is always the trigger for mail in flight.

## Prerequisites

- **Docker Desktop ≥ 24** with WSL2 backend
- **NVIDIA Container Toolkit** installed in WSL2 (`apt install nvidia-container-toolkit`)
- A GPU available to Docker (verify: `docker run --gpus all nvidia/cuda:12.4.0-base-ubuntu22.04 nvidia-smi`)

If you only need CPU inference, remove the `deploy.resources.reservations.devices` block from `docker-compose.yml`.

## First-run setup (one-time)

```powershell
# 1. Operator env: copy and fill the required values
cp .env.example .env
# Edit .env — at minimum set NURIKUN_UNSUB_SECRET (openssl rand -hex 32) +
# NURIKUN_SENDER_* if you'll use nurikun for outbound mail.

# 2. (nurikun + gmail only) Bootstrap Gmail OAuth on the HOST first.
#    OAuth consent needs a browser, so this can't run inside the container.
#    The resulting credentials.json + token.json are mounted into nurikun read-only.
mkdir oauth
# put your downloaded credentials.json at oauth/credentials.json, then:
.\musu-nurikun\musu-nurikun.exe gmail-token `
  --credentials .\oauth\credentials.json `
  --out .\oauth\token.json `
  --port 8765   # 8080 is taken by docker compose on bring-up

# 3. Build images + bring everything up
docker compose up -d --build

# 4. Pull a chat model into the ollama container (auto-cached in ollama_data)
docker compose exec ollama ollama pull llama3.2:1b
```

## Verify

```powershell
# All four services should be healthy / running
docker compose ps

# Ollama reachable from host
curl http://localhost:11434/v1/models

# crawl dashboard
start http://localhost:8080

# marketer REST API
curl http://localhost:8081/

# nurikun unsubscribe endpoint (signed; this will 400 without a valid sig)
curl http://localhost:8088/healthz   # → "ok"
```

## Common operations

```powershell
# View logs
docker compose logs -f nurikun
docker compose logs -f --tail=50

# Stop everything
docker compose down

# Stop + delete volumes (CAREFUL: loses ollama models, projects DBs, wiki)
docker compose down -v

# Rebuild a single service after code change
docker compose up -d --build crawl

# Run a one-shot command (e.g. nurikun doctor)
docker compose run --rm nurikun doctor --project default

# Run nurikun watch loop manually (since serve is the long-running mode)
docker compose run --rm nurikun watch --project default --limit 10

# Send a campaign (compose env carries the secrets through)
docker compose run --rm nurikun campaign send --list 1 --name "Weekly digest"
```

## Volumes (host paths)

| Host | Container | Purpose |
|---|---|---|
| `./wiki/` | `/wiki` (rw in crawl, ro in marketer) | Shared wiki state, bleve + vector indexes, harvested markdown |
| `./projects-marketer/` | `/app/projects` (marketer) | Per-project marketer state (campaigns DB, personas) |
| `./projects-nurikun/` | `/app/projects` (nurikun) | Per-project nurikun state (lists/subscribers DB) |
| `./oauth/` | `/oauth:ro` (nurikun) | Gmail credentials.json + token.json (operator-provisioned) |
| `ollama_data` (named) | `/root/.ollama` (ollama) | Cached models — keep this around |

`./projects-*` and `./oauth` are intentionally NOT in either repo's git — they hold per-deployment secrets and live state.

## Security notes

- `oauth/` is mounted **read-only** into nurikun. The Gmail token auto-refreshes via the refresh_token; nurikun never writes back to that file.
- `NURIKUN_UNSUB_SECRET` is required and unbounded — without it, signed unsubscribe links are not generated and `serve`'s `/unsubscribe` will reject every request. Treat it like a session key.
- All services run as the container's default user. Distroless/static base means no shell to exec into — by design.
- Outbound delivery (`nurikun watch` / `campaign`) is **NOT** part of the always-on `serve` daemon. They are one-shot via `docker compose run --rm nurikun ...` so a human (or scheduled cron) is always the trigger. This is intentional per nurikun's opt-in posture.

## Scheduling outbound campaigns

`nurikun watch` and `nurikun campaign send` are one-shot. To run periodically:

- **Windows Task Scheduler** (simplest on dev host): a daily task that runs `docker compose run --rm nurikun watch --project default --limit 50`.
- **Linux cron** on the deploy host.
- **A cron sidecar container** (`mcuadros/ofelia`, `willfarrell/crontab`) added to the compose file when you're ready.

I did not add a scheduler to the default compose — that's a deploy-policy decision, not a code one.

## Troubleshooting

- **`docker compose ps` shows ollama healthy but musu services restart** → check `docker compose logs <service>`; usually means an env var is missing (e.g. `NURIKUN_UNSUB_SECRET` empty causes nurikun to error on serve).
- **Ollama GPU not detected** → verify `docker run --gpus all nvidia/cuda:12.4.0-base-ubuntu22.04 nvidia-smi` works first. If not, fix WSL2 + nvidia-container-toolkit before debugging compose.
- **Gmail send fails** → token may have expired/been revoked. Re-run `gmail-token` bootstrap on the host, replace `oauth/token.json`, then `docker compose restart nurikun`.
- **Search returns "no index"** → run `docker compose run --rm crawl fetch web https://example.com --project default` (or any source) to populate the wiki.

## Production hardening (opt-in)

The default `docker compose up -d` is a clean dev bring-up. Production adds
four layers, all opt-in so dev stays simple.

### 1. Log rotation (always-on)

Already wired: every service caps its JSON log file at **10MB × 3 rotations
= 30MB max on disk**. The Docker default of unlimited growth is what eats
disk on long uptimes. Override via the `x-logging` anchor in
`docker-compose.yml` if you want different limits.

### 2. Scheduled `watch` (profile `scheduler`)

ofelia sidecar runs `nurikun watch` on a cron schedule. Edit
`ofelia/config.ini` to change interval / add `campaign send` jobs.

```powershell
docker compose --profile scheduler up -d
docker compose logs -f ofelia       # see what fires when
```

Default: `@every 10m`, `--limit 25`. Job runs via `exec` inside the already-
running `musu-nurikun` container, so env/volumes stay consistent with the
`serve` daemon. **Be careful enabling autonomous `campaign send`** — it
actually puts mail on the wire. The example in `ofelia/config.ini` is
intentionally commented out.

### 3. TLS termination via Caddy (profile `tls`)

Caddy sidecar terminates HTTPS in front of the musu services. Required
because Gmail/Outlook strip or quarantine plain-HTTP unsubscribe links in
campaign emails.

```powershell
# One-time:
mkdir caddy 2>$null
cp caddy/Caddyfile.example caddy/Caddyfile
# Edit caddy/Caddyfile — replace nurikun.example.com with your real hostname.
# Update .env: NURIKUN_PUBLIC_BASE_URL=https://your-hostname

docker compose --profile tls up -d
```

Requirements:
- Public DNS A/AAAA record for your hostname pointing at this host
- Ports 80 + 443 reachable from the public internet (Let's Encrypt HTTP-01 /
  TLS-ALPN-01 challenge)
- For local-only / dev: replace the hostname with `localhost` in Caddyfile
  and Caddy will issue a local self-signed cert (browser warning expected)

The example Caddyfile exposes only `nurikun`. `crawl` and `marketer`
blocks are commented out because their dashboards have no built-in auth —
gate them behind `basic_auth` or an SSO sidecar before exposing.

### 4. Pre-built images from GHCR (production overlay)

Instead of rebuilding from source on every deploy, pull tagged images
published by each repo's `.github/workflows/docker-publish.yml`:

```powershell
# Operator releases a version:
cd musu-nurikun
git tag -a v0.3.1 -m "release v0.3.1"
git push origin v0.3.1
# → GitHub Actions builds + pushes ghcr.io/yellowhama/musu-nurikun:v0.3.1
# Repeat for crawl-ai and marketer if their code changed.

# Then on the deploy host:
$env:MUSU_VERSION = "v0.3.1"
docker compose -f docker-compose.yml -f docker-compose.production.yml `
  --profile tls --profile scheduler up -d
```

The overlay (`docker-compose.production.yml`) replaces the `build:`
directives with `image: ghcr.io/yellowhama/musu-<name>:${MUSU_VERSION}`,
everything else (env / volumes / healthchecks / logging / depends_on)
stays the same.

First-time GHCR access from a private/locked-down host:
```bash
echo $GHCR_PAT | docker login ghcr.io -u <github-username> --password-stdin
```
(For yellowhama's own public packages, no auth needed for `docker pull`.)

### Putting it all together — full prod stack

```powershell
docker compose -f docker-compose.yml -f docker-compose.production.yml `
  --profile tls --profile scheduler up -d
docker compose logs -f                # tail everything
```

---

## Why this layout (design rationale)

- **Ollama isolated in its own container** so the user's `ollama serve` on the host (if any) doesn't conflict.
- **`crawl serve` is the dashboard, not a worker.** Actual harvesting (`fetch`, `research`) is one-shot. The compose just keeps the dashboard always-on for inspection.
- **`nurikun serve` is the public web layer** (unsubscribe/confirm). The autonomous loop (`watch` for inbound, `campaign` for outbound) is intentionally NOT a daemon — it's run on a schedule by the operator. This keeps the opt-in posture (a human always triggers any send).
- **MCP servers are NOT in compose.** MCP is stdio-only, spawned per-client (Claude Code, Cursor, etc.). Adding them to compose would be wrong — they need to be the client's child process.

---

This is the deploy bundle. The code itself ships from each repo's `Dockerfile`; the wiring lives in this directory.
