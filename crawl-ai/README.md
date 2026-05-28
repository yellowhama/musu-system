# musu-crawl-ai

> Agent-native knowledge harvesting and local wiki building.

`musu-crawl-ai` is the upstream knowledge system for the Musu stack. It fetches source material, normalizes it into a local wiki, keeps lexical and vector search surfaces warm, and exposes a CLI/MCP workflow that other agents can actually automate.

## What It Is

- A Go CLI for harvesting web, GitHub, Arxiv, Reddit, YouTube, and more
- A local wiki + index + vector store pipeline for grounded downstream agents
- A research tool that can plan, search, fetch, synthesize, and persist findings

## Best For

- building a project-specific LLM wiki
- collecting sources before drafting or support automation
- agent workflows that need durable local knowledge instead of one-off prompts

---

## 🚀 Key Features

### 🤖 Agent-Native Interface (v0.8.0)
- **MCP Server:** Native integration with Claude Desktop and Cursor. Use Musu as a built-in tool.
- **Machine-Readable:** Global `--json` mode for deterministic, noise-free output.
- **Agentic Recovery:** Error messages include `actionable_fix` tips to guide LLMs.

### 🧠 Intelligence & Research
- **Researcher Mindset:** Socratic planning, hypothesis testing, and contradiction detection.
- **Universal Harvesters:** YouTube (Transcript), Arxiv (HTML-first + OCR), GitHub, Reddit, and Web.
- **Multi-Modal:** Automatic image harvesting and local description (via LLaVA).
- **Live Sync:** Incremental per-fetch indexing — newly harvested documents are searchable immediately, without re-walking the filesystem or re-embedding the existing corpus.

### 🦾 Production-Grade Foundation
- **Thread-Safety:** Mutex-protected indexing for high-concurrency parallel crawls.
- **Secure Secrets:** Project-scoped credential management.
- **Visual Galaxy:** Interactive D3.js dashboard for knowledge mapping.

---

## 🛠️ Installation & Setup

### 1. Prerequisite: [Ollama](https://ollama.com)
Local LLM intelligence is required for Research, Vision, and Semantic Search.

### 2. Quick Start
```bash
./musu-crawl init --project my-research
./musu-crawl doctor --out ./wiki --project default
./musu-crawl doctor --out ./wiki --project default --fix
./musu-crawl fetch web https://go.dev/blog/go1.22 --project my-research
```

Core flow:
1. bootstrap a wiki/project
2. verify local readiness
3. fetch a source into the wiki
4. search, research, and reuse it downstream

### 3. MCP Integration (For Claude/Cursor)
Add the following to your `claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "musu": {
      "command": "C:/path/to/musu-crawl-ai/musu-crawl.exe",
      "args": ["mcp"]
    }
  }
}
```

### 4. MCP Server Registration (Claude Code CLI)

The MCP server inherits its environment from the registering process. Naive `claude mcp add` registrations end up running with default-only config (`localhost:11434/v1`, default model, etc.) — operators then wonder why `fetch`, `search`, and `research` are talking to the wrong endpoint or model.

Register with explicit `--env` flags so the server sees the same values your shell does:

```powershell
# Windows / PowerShell
claude mcp add -s user musu-crawl `
  -- musu-crawl.exe mcp `
  --env MUSU_AI_URL=http://localhost:11434/v1 `
  --env MUSU_AI_MODEL=llama3.2:1b `
  --env MUSU_AI_PROVIDER=ollama
```

```bash
# Linux / macOS
claude mcp add -s user musu-crawl \
  -- musu-crawl mcp \
  --env MUSU_AI_URL=http://localhost:11434/v1 \
  --env MUSU_AI_MODEL=llama3.2:1b \
  --env MUSU_AI_PROVIDER=ollama
```

Restart your Claude session after `claude mcp add` — tool schemas are read at session start. Verify with the `doctor` CLI (or the `fetch`/`search`/`research` MCP tools) returning results from your real endpoint, not the default fallback.

---

## 📖 Core Commands
- `fetch [source] [id]`: Harvest specific content.
- `research "[question]"`: Autonomous multi-agent deep research mission.
- `search "[query]"`: Local keyword and semantic vector search.
- `index --semantic`: Refresh the global knowledge graph and embeddings.
- `serve`: Launch the Galaxy Dashboard (Port 8080).
- `doctor`: Verify wiki/index presence and AI endpoint connectivity before long runs.

`doctor` also supports `--json`, which is useful when another agent or CI job needs deterministic preflight output.
`doctor --fix` can safely create a missing local wiki scaffold before longer runs.
Use `doctor --capability-source web --capability-source gh --capability-source yt` to get a machine-readable static source capability matrix during setup. This is capability metadata, not a live credential or reachability probe.
`init --json` now returns the wiki/project scaffold paths, AI reachability snapshot, and suggested next commands so bootstrap can be chained automatically.
JSON mode now uses the same top-level envelope as the other Musu tools: `status`, `message`, `data`, `actionable_fix`.
`actionable_fix` is now assembled from the specific failing checks, so missing wiki/project scaffolds and unreachable AI endpoints point to different recovery steps.
The command surface now has local smoke coverage for `fetch web ...` and `search ...`, so bootstrap and retrieval paths are verified without relying on live external services.
For a real endpoint-backed verification, set `MUSU_CRAWL_INTEGRATION_AI_URL` and run `go test -tags integration ./cmd`, or use `scripts/run-real-integration.ps1`.
Set `MUSU_CRAWL_INTEGRATION_EMBED_MODEL` when the reachable endpoint exposes an embedding model other than the default `nomic-embed-text`.
Set `MUSU_CRAWL_INTEGRATION_CHAT_MODEL` when the reachable endpoint exposes a chat model other than the default `llama3` for the real `research` smoke.
`research` can now use `MUSU_SEARCH_BASE_URL` for a deterministic local search harness instead of live DuckDuckGo during integration or lab verification.
The runner auto-probes `OLLAMA_HOST`, `127.0.0.1:11434`, and `localhost:11434`, checks both `/v1/models` and Ollama `/api/tags`, and prints explicit diagnostics when no reachable endpoint exists.
Use `-Json -ProbeOnly` when another agent or CI step needs machine-readable integration readiness output without actually running the integration-tag tests.
The JSON doctor now emits `issue_codes` such as `ollama_host_unspecified_bind_address`, `ollama_not_installed`, `localhost_probe_timeout`, and `missing_required_model`.

---

## 📂 Data Silos
Private research data is stored in the `wiki/` directory and is strictly excluded from Git tracking to ensure privacy.

---

## 🔗 The Ecosystem
- **musu-marketer:** The "Voice" that uses this knowledge for strategy.
- **musu-nurikun:** The "Hand" that works inboxes and opt-in mailing lists from grounded knowledge.
