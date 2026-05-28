# Next Steps: musu-crawl-ai

> 2026-05-28 audit items (MCP schemas, research/search guards, snake_case envelope, ignored Unmarshal sites, shared module extraction) are now CLOSED. See `MUSU_MCP_AUDIT_2026-05-28.md` and `MUSU_THERMONUCLEAR_REVIEW_2026-05-28.md`.

## P1
- keep `--capability-source` explicitly static unless a real live probe is implemented
- decide whether source readiness should ever become a live probe instead of documented static capability metadata

## P2
- add optional live probes for source credentials/network health
- split doctor reporting helpers from static capability metadata
- decide whether the local search harness seam (`MUSU_SEARCH_BASE_URL`) should stay integration-focused or become a documented alternative provider hook

## P3
- optimize index/vector rewrite cost for larger bulk crawls (current live-sync path is O(N) per doc / O(N²) per bulk crawl due to full index.json rewrites)
- improve ecosystem docs so the same entrypoint exists in all three repos
- ~~production hardening on the docker-compose bundle: TLS termination, log rotation, image registry push, scheduled batch crawls~~ — CLOSED 2026-05-28. Caddy `tls` profile (auto-HTTPS), x-logging anchor (10MB×3 rotation), `.github/workflows/docker-publish.yml` for GHCR push, ofelia `scheduler` profile available for crawl/research crons too (only nurikun-watch wired today; add `[job-exec "crawl-research"]` to `ofelia/config.ini` if desired).
- ~~first real GHCR push validation — operator pushes a `vX.Y.Z` tag and confirms the workflow publishes `ghcr.io/yellowhama/musu-crawl-ai:vX.Y.Z` + `:latest`~~ — VALIDATED 2026-05-28 against musu-marketer (`v2.0.4` published end-to-end with multi-arch amd64+arm64 + Trivy CRITICAL/HIGH scan + SARIF upload, ~8.4 min total). Same workflow shape applies here — first crawl-ai tag (`vX.Y.Z`) push will publish to `ghcr.io/yellowhama/musu-crawl-ai`.
- ~~README/AGENTS `claude mcp add --env ...` doc (F6 audit)~~ — CLOSED 2026-05-28 (`2500f3f`)
- ~~Trivy CRITICAL/HIGH scan + SARIF upload in docker-publish workflow~~ — CLOSED 2026-05-28 (`aafbc4c`, action `aquasecurity/trivy-action@v0.36.0`)

## Verified Integration Harness
- set `MUSU_CRAWL_INTEGRATION_AI_URL`
- optionally set `MUSU_CRAWL_INTEGRATION_EMBED_MODEL` (verified locally with `nomic-embed-text`)
- optionally set `MUSU_CRAWL_INTEGRATION_CHAT_MODEL` (verified locally with `llama3.2:1b`)
- run `go test -tags integration ./cmd`
- or run `powershell -ExecutionPolicy Bypass -File .\scripts\run-real-integration.ps1`
