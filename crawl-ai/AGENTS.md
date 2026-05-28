# 🤖 Guidance for AI Agents (Gemini CLI, Claude Code, etc.)

`musu-crawl-ai` is designed to be your primary high-performance data acquisition and knowledge management layer. If you are an AI agent "driving" this repository, follow these instructions to maximize your research performance.

## 🎓 Master Orchestration
For advanced multi-tool orchestration (Crawl + Marketer), refer to the **[MUSU_SKILL.md](../../MUSU_SKILL.md)** in the workspace root. Activating this skill transforms you into a Lead Orchestrator of the entire ecosystem.
This tool is a collection of high-quality **Knowledge Primitives**. You are encouraged to combine these primitives to perform complex, multi-step research.

### 1. The Knowledge Primitives
- **`fetch [source] [url]`**: Your primary tool to bypass blocks, clean HTML, and parse PDFs (including OCR for image-based PDFs if Tesseract is installed). Outputs clean Markdown with YAML.
- **`index`**: Re-generates the global `index.json` knowledge map and refreshes the Bleve/Vector search indexes.
- **`search [query]`**: High-performance search. Use `--semantic` for meaning-based retrieval if Ollama embeddings are available.
- **`doctor`**: Preflight the local wiki, index, project directory, and AI endpoint. Use `--json` for deterministic parsing, `--fix` for safe scaffold creation, and `--capability-source` when an agent needs static setup metadata for a source family.

## 🏎️ How to "Drive" this tool as an Agent

### Scenario: "Perform Multi-hop Deep Research"
1.  **Plan:** Decompose the goal into queries.
2.  **Discover:** Find URLs via search or direct inputs.
3.  **Harvest:** Run `.\musu-crawl fetch auto --file targets.txt -w 10 --project my_project`.
4.  **Analyze:** Read `wiki/index.json` and specific `.md` files in `wiki/projects/my_project/`.
5.  **Synthesize:** Use your own reasoning to answer, identifying gaps for the next "hop".

### Scenario: "Build a Persistent Knowledge Graph"
1.  After fetching, run `.\musu-crawl compile --project my_project`.
2.  This triggers the internal Compiler Agent (via Ollama) to link related nodes.
3.  **Manual Overwrite:** If Ollama is missing, you should manually read related files and write `[[WikiLinks]]` into the Markdown body yourself using file-edit tools.

### Scenario: "Manage Project Personas & Secrets"
1.  **Steer Yourself:** Write specific research instructions to `wiki/projects/{name}/PROMPT.md`.
2.  **Auth Management:** Run `.\musu-crawl auth set KEY VALUE --project {name}` to store project-scoped secrets (e.g., API keys).

## 🔌 MCP Server Registration

The MCP server inherits its environment from the registering process. Naive `claude mcp add` registrations end up running with default-only config (`localhost:11434/v1`, default model, etc.) — the `fetch`, `search`, and `research` MCP tools then talk to the wrong endpoint or model.

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

Restart your Claude session after `claude mcp add` — tool schemas are read at session start. Verify via the `doctor` CLI (or by invoking one of the MCP tools) that the server is hitting your real endpoint, not the default fallback.

## 🛑 Critical Mandates for Agents
- **Self-Healing:** If a command fails (e.g., missing Tesseract or Ollama), gracefully switch to your own logic (manual analysis or text-only extraction) and notify the user.
- **Preflight Honesty:** Treat `--capability-source` as static metadata, not as proof that an external source is reachable or authenticated.
- **Project Scoping:** Always use the `--project` flag to avoid context contamination.
- **Thread-Safe Ops:** Rely on the tool's built-in locking; do not worry about concurrent fetches corrupting the index.
- **Validation:** Always verify the structure of YAML Frontmatter before writing manual modifications to ensure `index.json` remains parsable.
