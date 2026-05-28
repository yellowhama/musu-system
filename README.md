# musu-core

Shared building blocks for the Musu agent ecosystem
([crawl-ai](https://github.com/yellowhama/musu-crawl-ai),
[marketer](https://github.com/yellowhama/musu-marketer),
[nurikun](https://github.com/yellowhama/musu-nurikun)).

## Packages

- **`env`** — three-layer config loader: process env > project `.env` >
  viper key. Same precedence rules for every CLI in the ecosystem.
- **`agent`** — OpenAI-compatible chat/embed/vision client with optional
  telemetry. One `Client` plus `Option`s (`WithVisionModel`, `WithTelemetry`,
  `WithRole`, `WithHTTPClient`) replaces the previously-triplicated
  `AgentClient` (~475 LOC of duplication removed).
- **`preflight`** — the AI-endpoint reachability `Probe` shared by every
  `doctor` command.

## Usage

```go
import (
    coreagent "github.com/yellowhama/musu-core/agent"
    coreenv "github.com/yellowhama/musu-core/env"
    corepreflight "github.com/yellowhama/musu-core/preflight"
)

client := coreagent.New(baseURL, model,
    coreagent.WithTelemetry(wikiDir, project),
)
reply, err := client.Ask("hi", false)

projectEnv := coreenv.LoadProjectEnv("projects/myproj/.env")
host := coreenv.String(viper, projectEnv, "imap_host", "MYAPP_IMAP_HOST")

if err := corepreflight.Probe(baseURL); err != nil { /* AI unreachable */ }
```

## Tests

```
go test ./...
```

All three packages have table-driven test coverage with no network calls
(httptest for the agent client and probe). No external services required.
