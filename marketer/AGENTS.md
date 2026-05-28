# 🤖 Guidance for AI Agents (musu-marketer)

`musu-marketer` is your "Voice." Use this tool to transform verified knowledge into high-impact marketing campaigns.

## 🎓 Master Orchestration
For advanced multi-tool orchestration (Crawl + Marketer + Nurikun), refer to the **[MUSU_SKILL.md](../../MUSU_SKILL.md)** in the workspace root. Activating this skill transforms you into a Lead Orchestrator of the entire ecosystem.

## 🏗️ Core Architecture for Agents
This tool provides **Marketing Primitives** grounded in professional strategy.

### 1. The Marketing Primitives
- **`doctor`**: Run this first. It checks the wiki path, project scaffold, DB, AI endpoint, and optional topic readiness. Use `--json` for deterministic agent parsing and `--fix` when the local project scaffold is missing.
- **`draft [topic]`**: The main content-generation step. It triggers the Strategist (STP analysis) and then the Copywriter (content creation).
- **`autopilot [topic]`**: A fuller orchestration path that assumes research material already exists or can be prepared in the wider Musu workflow.
- **`persona [list/create/show]`**: Manage the brand's identity and tone of voice.
- **`publish [ID]`**: Pushes drafts to registered publisher adapters (for example `local` or `webhook`).

## 🤝 Ecosystem Collaboration: The "Influence Pipeline"
*Scenario: An agent acts as a full-service marketing agency using the Musu team.*

1.  **SPOT TRENDS:** Use `musu-crawl spot --json` to find what's hot in a specific niche.
2.  **DEEP RESEARCH:** Use `musu-crawl research --json` to gather verified facts about that trend into a Wiki.
3.  **PREFLIGHT:** Use `musu-marketer doctor --project <name> --topic "<topic>" --json` to verify the wiki and AI endpoint are actually ready for the chosen draft.
4.  **STRATEGIZE:** Use `musu-marketer draft` to generate a high-impact campaign based on that Wiki data.
5.  **DELIVER:** Use `musu-marketer publish` to output the final copy, or pass the campaign into `musu-nurikun` for compliant opt-in email delivery.

## 🔌 MCP Server Registration

The MCP server inherits its environment from the registering process. Naive `claude mcp add` registrations end up running with default-only config (`localhost:11434/v1`, default model, no wiki override) — the `draft_campaign` and `list_campaigns` MCP tools then talk to the wrong endpoint or look at an empty wiki.

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

Restart your Claude session after `claude mcp add` — tool schemas are read at session start. Verify by invoking `list_campaigns` and confirming it returns your real project state, not an empty default scaffold.

## 🛑 Critical Mandates for Agents
- **Strategy First:** Always ensure a `Strategic Brief` is generated before outputting content.
- **Bible Compliance:** Strictly follow the frameworks in the `MARKETING_BIBLE.md`.
- **Grounding First:** If `doctor --topic` fails, do not bluff. Improve the crawl wiki and re-run preflight.
- **Self-Healing:** If the wiki path is wrong, update `config.yaml`, use `--wiki`, or rely on the sibling auto-discovery path.
