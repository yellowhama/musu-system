# musu-nurikun

> **The Autonomous Email Agent — Inbound Support & Compliant Opt-in Outreach.**

`musu-nurikun` is the "Hand" of the Musu ecosystem. It works a service mailbox you
own: triaging and answering inbound customer email, and sending **opt-in** mailing-list
campaigns at a controlled cadence. It only ever emails people who subscribed
themselves — there is no cold outreach, no scraped lists, no fake identities, and no
anti-spam evasion.

---

## 🚀 What it does

### 📥 Inbound Support
- **Triage:** Classifies each incoming message (category, intent, language, urgency).
- **Grounded replies:** Answers from a configurable knowledge source, not model guesswork.
- **Confidence-gated sending:** High-confidence, allow-listed replies send automatically;
  anything uncertain or sensitive (refunds, complaints, legal) is drafted and **escalated to a human**.

### 📤 Opt-in Outreach (newsletters / sales)
- **Self-subscribe + double opt-in:** A subscriber signs up, confirms via a verification
  link, and only then becomes `confirmed`. Consent is provable.
- **Controlled cadence:** Per-list frequency caps (e.g. 1–2 emails / week / subscriber).
- **Compliance built in:** `(광고)` subject labeling (정보통신망법 §50), sender identification,
  one-click unsubscribe, and a hard **suppression list** gate at send time.
- **Per-domain rate limiting** to protect deliverability and sender reputation.

---

## 🧩 Pluggable by design (ship-anywhere)

- **Mailbox** — `IMAP/SMTP` (any provider) or `Gmail API`, selected by config.
- **Knowledge source** — `musu-crawl-ai` wiki (RAG), a local `FAQ/docs` folder, or none.
- **AI** — any OpenAI-compatible endpoint via `ai_url` (defaults to local Ollama `/v1`).

Drop it into any project: point it at a mailbox, choose a knowledge source, and go.

---

## 🛠️ Quick Start

```bash
./musu-nurikun init --project acme-support
# configure mailbox + knowledge source in projects/acme-support/config.yaml
./musu-nurikun watch        # inbound: triage + reply per policy
./musu-nurikun campaign send weekly-digest   # outbound: opt-in subscribers only
```

> Requires an OpenAI-compatible AI endpoint (e.g. [Ollama](https://ollama.com)) for triage/reply.

---

## 📂 Data & Privacy

Lists, subscribers, consent records, message threads, and suppression lists live in the
`projects/` directory (SQLite) and are excluded from Git. Consent state and unsubscribes
are authoritative — the agent cannot email a suppressed address.

---

## 🔗 The Ecosystem

- **musu-crawl-ai:** The "Brain" — harvests the product knowledge replies are grounded in.
- **musu-marketer:** The "Voice" — supplies the brand persona and campaign copy.
