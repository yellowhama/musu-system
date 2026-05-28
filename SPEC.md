# musu-nurikun — Spec (STATUS: v0.3.0 EMAIL AGENT)

## 🎯 Project Goal
An autonomous **email agent** that works a service mailbox you own: triaging and
answering inbound customer support, and sending **opt-in** mailing-list campaigns
at a controlled cadence. It is the "Hand" of the Musu ecosystem.

It emails only people who subscribed themselves (double opt-in). There is no cold
outreach, no scraped lists, no fabricated identities, and no anti-detection — the
prior "digital citizen / stealth signup" design (≤ v0.2.x) was removed in the
v0.3.0 pivot.

## ✅ Milestones

### v0.3.0 — Email-agent pivot
- [x] **Removed** the fake-identity / anti-detection stack (signup automation,
      fingerprint spoofing, human-mouse mimicry, disposable mail, persona forging).
- [x] **Contracts:** `Mailbox` (IMAP/SMTP + Gmail), `knowledge.Source`
      (crawl-ai | folder | none), `policy` send-engine, project `config`, and a
      SQLite schema for lists / subscribers / messages / campaigns / suppression.
- [x] **Inbound:** `watch` — fetch → triage (LLM) → ground in knowledge → draft →
      policy decides: auto-send high-confidence allow-listed replies, else escalate
      or draft for review. `reply` to act on a stored draft.
- [x] **Outbound (opt-in):** `lists`, `subscribe` (pending), `confirm`
      (double opt-in), `campaign` (sends to confirmed, non-suppressed, due
      subscribers), `contacts`, `suppress`.
- [x] **Compliance (non-bypassable):** `(광고)` subject label (정보통신망법 §50),
      sender + postal footer, RFC 8058 `List-Unsubscribe`, hard suppression gate,
      per-domain rate limit.
- [x] **Web endpoints:** `serve` exposes HMAC-signed one-click `/unsubscribe` and
      double opt-in `/confirm`.

### v0.3.x — Hardening + live verification
- [x] Removed dead `firstNonEmpty` helper (superseded by env-aware `valueString`/`valueInt`).
- [x] Telemetry `logTrace` I/O errors surfaced to stderr (no silently-lost traces).
- [x] Compiled `musu-nurikun.exe` no longer tracked in git (already gitignored).
- [x] **Live HTTP roundtrip verified** — `serve` + HMAC-signed `/unsubscribe` + web `/confirm` + tamper rejection (HTTP 400) all end-to-end verified. Externally-computed (openssl) HMAC signatures interoperate with `compliance.SignUnsub`, confirming the RFC 8058 one-click unsubscribe surface is real and not self-referential.
- [x] **Shared module integration**: `internal/agent/client.go`, `internal/config/config.go`, `internal/preflight/doctor.go` now thin wrappers over `github.com/yellowhama/musu-core/{agent,env,preflight}` v0.1.0 (689 LOC ecosystem-wide deduplication, see `project_musu_core_shared_module`).
- [x] **`gmail-token` bootstrap command** — one-off OAuth consent → `token.json` (refresh_token included) with local loopback callback. Closes the manual OAuth provisioning friction; rundll32-based browser open avoids cmd.exe `&` URL truncation.
- [x] **Live Gmail API verified** — token.json successfully authenticated against real Gmail (`gmail.users.getProfile → ceo@yellowhama.com`).
- [x] **MCP server exposure** — `cmd/mcp.go` exposes 8 safe ops as MCP tools (doctor, list_lists, create_list, subscribe, confirm_subscriber, list_subscribers, suppress, messages_by_status); delivery ops (`watch`, `campaign`) intentionally CLI-only.
- [x] **MCP tool schemas declared** — `WithString`/`WithNumber`/`Required`/`Enum` properly attached (was empty schema, blocking client args).
- [x] **`db.NewStore` MkdirAll(parent)** — cwd-isolated invocations (e.g. MCP servers) no longer fail with SQLITE_CANTOPEN on missing project dirs.
- [x] **`preflight.DoctorResult` JSON envelope** — snake_case `json` tags so MCP envelope is consistent with the inner Report.
- [x] **Docker deploy bundle** — Dockerfile (alpine + digest-pinned golang) committed; brings up under the top-level docker-compose with crawl/marketer/ollama via shared volumes. End-to-end `compose up` verified healthy.

## 🔌 Configurable per deployment (ship-anywhere)
- **Mailbox** — `imap` (IMAP fetch + SMTP send) or `gmail` (API), via `mailbox_provider`.
- **Knowledge** — `crawlai` (RAG over a musu-crawl-ai wiki), `folder` (local docs), or `none`.
- **AI** — any OpenAI-compatible endpoint via `ai_url` (default local Ollama `/v1`).

## ⚠️ Operational notes
- **Gmail** requires an operator-provisioned OAuth client JSON + a cached token
  with a refresh token; the agent does not run an interactive consent flow.
- **IMAP** uses implicit TLS (typically :993); **SMTP** uses STARTTLS (typically :587).
- One-click unsubscribe links are HMAC-signed (`unsub_secret`); set `public_base_url`
  so campaigns embed links pointing at `serve`. Without them, campaigns fall back to a
  `mailto:` unsubscribe.
- `doctor --fix` now accepts the same preset knobs as `init`:
  - `--mailbox-provider imap|gmail`
  - `--knowledge-source none|crawlai|folder`
  This keeps project bootstrap semantics aligned between first-time setup and scaffold recovery.
- Project-local `.env` secrets are now supported in addition to `config.yaml`, and `init` writes:
  - `.env.example`
  - `bootstrap.ps1`
  - Gmail `oauth/README.md` for credential/token placement

## 🚧 Not yet done / future
- Live send/receive verification against a real mailbox.
- A confirm/unsubscribe landing-page UI (currently plain-text responses).
- Outbound campaign personalization via a shared `musu-marketer` persona.
- More guided mailbox bootstrap / OAuth setup UX.

---
**Status:** 📬 EMAIL AGENT (v0.3.0)
