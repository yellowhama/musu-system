# musu-marketer

> Grounded campaign drafting from a verified local wiki.

`musu-marketer` is the strategy and writing layer of the Musu stack. It consumes grounded source material from `musu-crawl-ai`, turns it into campaign briefs and finished copy, and keeps the whole flow deterministic enough for CLI automation, CI smoke tests, and agent handoff.

## What It Is

- A Go CLI for drafting marketing campaigns from a grounded wiki
- Persona-aware copy generation with a copywriter/critic loop
- A project-local campaign database with machine-readable output and preflight checks

## Best For

- product launch messaging
- newsletter and blog campaign drafting
- grounded marketing copy where source material matters more than vibes

---

## 🚀 Key Features

### 🎭 The Strategic Crew (v2.0.0)
- **Copywriter-Critic Loop:** Every draft is audited by a Zero-Tolerance Critic agent to eliminate AI fluff and maximize impact.
- **Social Memory:** Learns from past published successes to maintain narrative consistency.
- **Marketing Bible:** Injects 10+ years of professional marketing frameworks (AIDA, PAS, BAB) into the core logic.

### 🤖 Native Integration
- **MCP Server:** Native tool discovery for Claude and Gemini.
- **REST API Server:** Lightweight HTTP interface for backend product integration.
- **Interactive Builder:** Wizard-based persona creation (`persona create`).

### 🦾 Production Hardening
- **Resource Efficient:** Optimized HTTP connection pooling and singleton clients.
- **Project Siloing:** Strictly isolated assets, databases, and identities per mission.
- **Embedded Marketing Bible:** Core strategy prompts no longer depend on the shell's current working directory.

---

## 🛠️ Installation & Setup

### 1. Prerequisites
- **Data Source:** Requires [musu-crawl-ai](https://github.com/yellowhama/musu-crawl-ai) wiki.
- **Intelligence:** Requires [Ollama](https://ollama.com).

### 2. Quick Start
```bash
./musu-marketer init --project master-brand
./musu-marketer doctor --project master-brand
./musu-marketer draft "QuantumComputing" --persona tech-analyst
```

Core flow:
1. bootstrap a project
2. verify wiki + AI readiness
3. draft against verified source material

### 3. MCP Server Registration (Claude Code CLI)

The MCP server inherits its environment from the registering process. Naive `claude mcp add` registrations end up running with default-only config (`localhost:11434/v1`, default model, no wiki override) — operators then wonder why `draft_campaign` and `list_campaigns` are talking to the wrong endpoint or pointing at an empty wiki.

Register with explicit `--env` flags so the server sees the same values your shell does:

```powershell
# Windows / PowerShell
claude mcp add -s user musu-marketer `
  -- musu-marketer.exe mcp `
  --env MUSU_AI_URL=http://localhost:11434/v1 `
  --env MUSU_AI_MODEL=llama3.2:1b `
  --env MUSU_WIKI_DIR=$env:USERPROFILE\musu-crawl-ai\wiki
```

```bash
# Linux / macOS
claude mcp add -s user musu-marketer \
  -- musu-marketer mcp \
  --env MUSU_AI_URL=http://localhost:11434/v1 \
  --env MUSU_AI_MODEL=llama3.2:1b \
  --env MUSU_WIKI_DIR=$HOME/musu-crawl-ai/wiki
```

Restart your Claude session after `claude mcp add` — tool schemas are read at session start. Verify by calling the `list_campaigns` MCP tool and confirming it surfaces your real project, not an empty default scaffold.

---

## 📖 User Manual

### 1. Strategic Drafting
Generate content using a self-correcting loop:
```bash
./musu-marketer draft [topic] --persona [name] --project [project]
```

### 0. Preflight
Verify that your crawl wiki and AI endpoint are reachable before drafting:
```bash
./musu-marketer doctor --project [project]
./musu-marketer doctor --project [project] --json
./musu-marketer doctor --project [project] --fix
./musu-marketer doctor --project [project] --topic "your-topic"
```

`musu-marketer` now auto-discovers a sibling `../musu-crawl-ai/wiki` when `--wiki` is omitted. Override it with `--wiki`, `MUSU_MARKETER_WIKI`, or `MUSU_WIKI` when needed.
When safe, `doctor --fix` will scaffold the missing local project directory/database/config for you.
If you pass `--topic`, `doctor` checks whether the current wiki contains enough matching source material to draft that topic by ranking `index.json` metadata plus Markdown body evidence, not just filenames.
`init --json` now returns structured bootstrap details (`project_dir`, `db_path`, `config_path`, `next_steps`) so other agents can continue setup without scraping human logs.
For smoke checks, a tiny grounded wiki fixture now lives under `examples/sample-wiki/`.
Like the other Musu tools, JSON mode now uses the same top-level envelope: `status`, `message`, `data`, `actionable_fix`.
`actionable_fix` is derived from the actual failing checks, so AI-unreachable, missing-project, and topic-grounding failures produce different next steps.
The command surface now has local smoke coverage for the real `draft` path, not just the lower-level strategist/copywriter helpers.
For a real endpoint-backed verification, set `MUSU_MARKETER_INTEGRATION_AI_URL` and run `go test -tags integration ./cmd`, or use `scripts/run-real-integration.ps1`.
Set `MUSU_MARKETER_INTEGRATION_MODEL` when the reachable endpoint exposes a chat model other than the default `llama3`.
The topic bridge now prefers stronger title/tag/summary hits ahead of weak body-only matches, so drafts start from better-grounded sources when the wiki is noisy.
The runner auto-probes `OLLAMA_HOST`, `127.0.0.1:11434`, and `localhost:11434`, checks both `/v1/models` and Ollama `/api/tags`, and prints explicit diagnostics when no reachable endpoint exists.
Use `-Json -ProbeOnly` when another agent or CI step needs machine-readable integration readiness output without actually running the integration-tag tests.
The JSON doctor now emits `issue_codes` such as `ollama_host_unspecified_bind_address`, `ollama_not_installed`, `localhost_probe_timeout`, and `missing_required_model`.

### 2. Autopilot (Zero-Click)
Spot trends via `crawl-ai`, research them, and draft campaigns automatically:
```bash
./musu-marketer autopilot [subreddit]
```

### 3. Campaign Management
```bash
./musu-marketer list
./musu-marketer view [ID]
./musu-marketer publish [ID] --platform local
```

---

## 📂 Architecture
All campaign data and project-specific personas are stored in the `projects/` directory, which is excluded from Git to protect your marketing secrets.

---

## 🔗 The Ecosystem
- **musu-crawl-ai:** The "Brain" providing verified knowledge.
- **musu-nurikun:** The "Hand" handling inbox triage and compliant opt-in email operations.
