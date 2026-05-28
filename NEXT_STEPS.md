# Next Steps: musu-nurikun

## P1
- improve mailbox/OAuth bootstrap UX
- turn the sample configs into a more guided setup path or wizard

## P2
- add a real mailbox/bootstrap smoke path or staged operator checklist around Gmail/IMAP credentials
- deepen marketer persona integration for campaign tone consistency

## P3
- add Gmail token/bootstrap operator docs and a tighter first-run checklist around the sample configs
- extract shared module(s) for `AgentClient` + `preflight/doctor` + env-loader to remove triple-duplicated logic across the three repos
- run a live mailbox roundtrip (real IMAP/Gmail + Ollama) to close the only remaining "untested in production" gate

## Verified Integration Harness
- set `MUSU_NURIKUN_INTEGRATION_AI_URL`
- optionally set `MUSU_NURIKUN_INTEGRATION_MODEL` (verified locally with `llama3.2:1b`)
- run `go test -tags integration ./cmd`
- or run `powershell -ExecutionPolicy Bypass -File .\scripts\run-real-integration.ps1`
