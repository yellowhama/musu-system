# 🤖 Guidance for AI Agents (musu-nurikun)

`musu-nurikun` is the "Hand" of the Musu ecosystem: an autonomous **email agent**
for a service mailbox you own. It does inbound customer support and **opt-in**
mailing-list outreach. It never creates fake identities, scrapes lists, sends cold
mail, or evades spam detection — emailing only self-subscribed, confirmed recipients.

## 🏗️ Action Primitives
Inbound:
- **`doctor`** — run first. It checks project config, mailbox settings, knowledge source wiring, AI endpoint reachability, and can safely recreate a missing scaffold with `--fix`.
- **`watch`** — fetch inbound mail, triage it, draft a grounded reply, then auto-send
  high-confidence allow-listed replies or escalate/draft the rest for a human.
- **`reply --id N [--send]`** — print or send a stored drafted/escalated reply.

Outbound (opt-in only):
- **`lists`** — create / list mailing lists (with a per-list send cadence).
- **`subscribe --list N --email E`** — record a *pending* subscriber (not yet mailable).
- **`confirm --token T`** — complete double opt-in (also available as a `/confirm` link via `serve`).
- **`campaign --list N --subject .. --body ..`** — send to confirmed, non-suppressed,
  due subscribers, with mandatory `(광고)` labeling, footer, `List-Unsubscribe`, and rate limiting.
- **`contacts --list N`** / **`suppress --email E`** — inspect subscribers / unsubscribe.
- **`serve`** — HMAC-signed one-click `/unsubscribe` and `/confirm` web endpoints.

## 🔌 Configuration
Per project `config.yaml` or env (`NURIKUN_*`): `mailbox_provider` (imap|gmail) + creds,
`knowledge_source` (crawlai|folder|none), `ai_url` (OpenAI-compatible), sender identity,
`public_base_url` + `unsub_secret` for signed unsubscribe links.

Bootstrap symmetry matters:
- `init --mailbox-provider ... --knowledge-source ...` sets the initial config shape.
- `doctor --fix --mailbox-provider ... --knowledge-source ...` now uses the same presets when recreating a missing scaffold.

## 🤝 Ecosystem Collaboration
1. **KNOWLEDGE** — `musu-crawl-ai` harvests product docs/FAQ; set `knowledge_source=crawlai`
   so replies are grounded in it.
2. **VOICE** — reuse a `musu-marketer` persona for consistent tone in replies/campaigns.
3. **HAND** — `musu-nurikun` triages inbound and sends opt-in campaigns.

## 🔌 MCP Server Registration

The MCP server inherits its environment from the registering process. Naive `claude mcp add` registrations end up running with default-only config (`localhost:11434/v1`, no Gmail creds, no public base URL, etc.) — `doctor` then reports everything missing.

Register with explicit `--env` flags so the server sees the same secrets your shell does:

```powershell
# Windows / PowerShell
claude mcp add -s user musu-nurikun `
  -- musu-nurikun.exe mcp `
  --env NURIKUN_AI_URL=http://localhost:11434/v1 `
  --env NURIKUN_AI_MODEL=llama3.2:1b `
  --env NURIKUN_MAILBOX_PROVIDER=gmail `
  --env NURIKUN_GMAIL_CREDENTIALS=$env:USERPROFILE\.config\nurikun\credentials.json `
  --env NURIKUN_GMAIL_TOKEN=$env:USERPROFILE\.config\nurikun\token.json `
  --env NURIKUN_KNOWLEDGE_SOURCE=crawlai `
  --env NURIKUN_PUBLIC_BASE_URL=https://your.domain.example `
  --env NURIKUN_UNSUB_SECRET=$env:NURIKUN_UNSUB_SECRET `
  --env NURIKUN_SENDER_NAME="Your Sender Name" `
  --env NURIKUN_SENDER_ADDRESS=you@yourdomain.com `
  --env NURIKUN_SENDER_PHYSICAL="Your mailing address"
```

```bash
# Linux / macOS
claude mcp add -s user musu-nurikun \
  -- musu-nurikun mcp \
  --env NURIKUN_AI_URL=http://localhost:11434/v1 \
  --env NURIKUN_AI_MODEL=llama3.2:1b \
  --env NURIKUN_MAILBOX_PROVIDER=gmail \
  --env NURIKUN_GMAIL_CREDENTIALS=$HOME/.config/nurikun/credentials.json \
  --env NURIKUN_GMAIL_TOKEN=$HOME/.config/nurikun/token.json \
  --env NURIKUN_KNOWLEDGE_SOURCE=crawlai \
  --env NURIKUN_PUBLIC_BASE_URL=https://your.domain.example \
  --env NURIKUN_UNSUB_SECRET=$NURIKUN_UNSUB_SECRET \
  --env NURIKUN_SENDER_NAME="Your Sender Name" \
  --env NURIKUN_SENDER_ADDRESS=you@yourdomain.com \
  --env NURIKUN_SENDER_PHYSICAL="Your mailing address"
```

Restart your Claude session after `claude mcp add` — tool schemas are read at session start. Verify by calling the `doctor` MCP tool and confirming it returns your real config (mailbox provider, public base URL, sender identity) — not the default fallback.

The MCP server exposes 8 safe ops: `doctor`, `list_lists`, `create_list`, `subscribe`, `confirm_subscriber`, `list_subscribers`, `suppress`, `messages_by_status`.

> **Note:** delivery ops (`watch`, `campaign`, `serve`) are intentionally CLI-only — keeping anything that actually puts mail on the wire in human-in-the-loop territory. This is part of the opt-in posture; **never** wrap them as MCP tools.

## 🛑 Critical Mandates
- **Opt-in only.** Never email an address that is not `confirmed`. The suppression list
  is authoritative and is re-checked at send time; never bypass it.
- **Compliance is not optional.** `(광고)` label, sender/postal footer, and one-click
  unsubscribe are applied to every campaign message — there is no flag to skip them.
- **Never reintroduce** identity forging, fingerprint/anti-detection, or cold outreach.
- **Privacy:** never leak the contents of `projects/` (subscribers, consent, threads).
