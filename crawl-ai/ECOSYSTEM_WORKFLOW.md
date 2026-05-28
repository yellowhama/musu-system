# Musu Ecosystem Workflow

This is the practical order for using the three Musu tools together.

1. `musu-crawl-ai` — harvest knowledge into a local wiki
2. `musu-marketer` — read that wiki and draft campaigns/persona-shaped copy
3. `musu-nurikun` — use the knowledge/persona in inbound support and compliant opt-in email

## 1. Crawl first

```bash
cd F:/Aisaak/Projects/musu-crawl-ai
./musu-crawl.exe init --out ./wiki --project product-research
./musu-crawl.exe doctor --out ./wiki --project default
./musu-crawl.exe doctor --out ./wiki --project default --fix
./musu-crawl.exe doctor --out ./wiki --project product-research --capability-source web --capability-source gh --capability-source yt --json
./musu-crawl.exe fetch web https://example.com --project product-research
./musu-crawl.exe index --out ./wiki --project product-research
```

What this gives you:
- markdown wiki pages
- `index.json`
- `musu.bleve`
- optional vectors for semantic search

## 2. Draft from the wiki

`musu-marketer` can auto-discover a sibling `../musu-crawl-ai/wiki`, but you can override with `--wiki`.

```bash
cd F:/Aisaak/Projects/musu-marketer
./musu-marketer.exe init --project launch
./musu-marketer.exe doctor --project launch
./musu-marketer.exe doctor --project launch --fix
./musu-marketer.exe doctor --project launch --topic "AI farmland analysis"
./musu-marketer.exe draft "AI farmland analysis" --persona default --project launch
./musu-marketer.exe list --project launch
./musu-marketer.exe publish 1 --platform local --project launch
```

Blocking failures usually mean:
- the wiki path is wrong
- the project has not been initialized
- the OpenAI-compatible endpoint at `--ai-url` is not reachable

## 3. Operate the inbox or opt-in list

`musu-nurikun` uses a project-scoped `config.yaml`. `init` creates a commented template.

```bash
cd F:/Aisaak/Projects/musu-nurikun
./musu-nurikun.exe init --project acme-support --mailbox-provider imap --knowledge-source crawlai
./musu-nurikun.exe doctor --project acme-support
./musu-nurikun.exe doctor --project acme-support --fix
./musu-nurikun.exe watch --project acme-support
./musu-nurikun.exe campaign --list 1 --name weekly-digest --subject "업데이트" --body "..." --project acme-support
```

Important:
- if `doctor` fails, fix mailbox config first
- `knowledge_source: crawlai` should point to the `musu-crawl-ai` wiki
- `public_base_url` and `unsub_secret` should be configured before real campaigns

## Current reality / caveats

- `musu-crawl-ai` is the most immediately usable of the three
- `musu-marketer` is usable, but wiki path and AI endpoint must be right
- `musu-nurikun` is now aligned with the current source, but it still depends on correct mailbox/bootstrap config

## Machine-readable preflight

All three tools now support a preflight `doctor` command.

- `musu-crawl-ai doctor --json`
- `musu-marketer doctor --json`
- `musu-nurikun doctor --json`

Use those in scripts or CI to detect setup drift before longer runs.
