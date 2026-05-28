# musu-crawl-ai: Autonomous Researcher Rules

As an autonomous agent driving this project, follow these core mandates:

## 🏎️ Driving Strategy
- **Self-Verification:** Always run `go build` and `go fmt` after modifying source code.
- **Error Loop:** If a command fails, analyze the stderr, fix the code, and retry immediately without asking.
- **Wiki Maintenance:** Every time a new harvester is added or modified, run `./musu-crawl index` to ensure the knowledge base is consistent.

## 🛠️ Standards
- **Go Idioms:** Use standard Go 1.21+ patterns. Keep dependencies minimal.
- **Documentation:** Keep `SPEC.md` and `README.md` updated with every feature change.
- **RAG Readiness:** Ensure `index.json` and `musu.bleve` are always healthy.

## 🛑 Safety & Boundaries
- Do not delete the `wiki/` directory contents unless explicitly requested.
- Protect `.git` and `go.mod` integrity.
