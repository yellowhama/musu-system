# Next Steps: musu-marketer

> 2026-05-28 audit items (MCP schemas, draft guard, db MkdirAll, snake_case envelope, shared module extraction) are now CLOSED. See `MUSU_MCP_AUDIT_2026-05-28.md` and `MUSU_THERMONUCLEAR_REVIEW_2026-05-28.md`.

## P1
- decide whether ranked lexical topic retrieval should stay simple or move toward semantic/vector retrieval
- extract doctor reporting/fixing helpers from `cmd/doctor.go`

## P2
- extend the sample wiki into a richer multi-topic fixture
- expose topic-readiness explanations more explicitly in JSON output

## P3
- improve publish adapters beyond local/webhook
- ~~production hardening on the docker-compose bundle: TLS termination, log rotation, image registry push~~ — CLOSED 2026-05-28. Caddy `tls` profile (auto-HTTPS), x-logging anchor (10MB×3 rotation), `.github/workflows/docker-publish.yml` for GHCR push.
- ~~first real GHCR push validation — operator pushes a `vX.Y.Z` tag and confirms the workflow publishes `ghcr.io/yellowhama/musu-marketer:vX.Y.Z` + `:latest`~~ — CLOSED 2026-05-28. `v2.0.2` proved the publish path; `v2.0.3` caught a wrong Trivy action pin (`0.24.0` → `v0.36.0`); `v2.0.4` is the green end-to-end reference run (Build 463s + Trivy 10s + SARIF 6s, multi-arch amd64+arm64, `ghcr.io/yellowhama/musu-marketer:v2.0.4` + `:latest` pullable, binary reports v2.0.4).
- ~~README/AGENTS `claude mcp add --env ...` doc (F6 audit)~~ — CLOSED 2026-05-28 (`ac03064`)
- ~~Trivy CRITICAL/HIGH scan + SARIF upload in docker-publish workflow~~ — CLOSED 2026-05-28 (`eca4074`, action `aquasecurity/trivy-action@v0.36.0`)

## Verified Integration Harness
- set `MUSU_MARKETER_INTEGRATION_AI_URL`
- optionally set `MUSU_MARKETER_INTEGRATION_MODEL` (verified locally with `llama3.2:1b`)
- run `go test -tags integration ./cmd`
- or run `powershell -ExecutionPolicy Bypass -File .\scripts\run-real-integration.ps1`
