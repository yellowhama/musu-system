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

## 🛑 Critical Mandates
- **Opt-in only.** Never email an address that is not `confirmed`. The suppression list
  is authoritative and is re-checked at send time; never bypass it.
- **Compliance is not optional.** `(광고)` label, sender/postal footer, and one-click
  unsubscribe are applied to every campaign message — there is no flag to skip them.
- **Never reintroduce** identity forging, fingerprint/anti-detection, or cold outreach.
- **Privacy:** never leak the contents of `projects/` (subscribers, consent, threads).
