# musu-nurikun

> **The Autonomous Digital Citizen & Action Layer.**

`musu-nurikun` (무수-누리꾼) is a high-performance AI agent designed to exist autonomously on the web. It is the "Hand and Feet" of the Musu ecosystem, responsible for acquiring digital identities, registering on platforms, and interacting with communities with human-like stealth.

---

## 🚀 Key Features

### 🕵️ Perfect Disguise (v0.2.1)
- **Behavioral Stealth:** Mimics human mouse movements using **Cubic Bezier Curves**, easing, and random jitter.
- **Hardware Spoofing:** Spoofs WebGL hardware (NVIDIA) and injects Canvas noise to defeat fingerprinting.
- **Dynamic Identity:** Randomizes User-Agents and viewports for every session.

### 🐣 Autonomous Birth
- **Cognitive Navigator:** Uses LLM-driven (Ollama) DOM analysis to handle complex, dynamic signup forms without hardcoded scripts.
- **Identity Forger:** Generates verifiable personas and manages them in a persistent SQLite database.
- **SMS/Email Ready:** Pluggable interface for real-world verification services (SMSPool, AgentMail).

### 🦾 Hardened Execution
- **Resource Managed:** Pooled HTTP connections and strict browser context lifecycle management.
- **Project Isolation:** Isolated browser profiles, cookies, and local storage per mission.

---

## 🛠️ Installation & Setup

### 1. Prerequisites
- **Browsers:** Requires [Playwright](https://playwright.dev/) dependencies.
- **Intelligence:** Requires [Ollama](https://ollama.com).

### 2. Quick Start
```bash
./musu-nurikun init
./musu-nurikun forge "JohnDoe"
./musu-nurikun signup reddit --id 1
```

---

## 📖 User Manual

### 1. Forging an Identity
Create a new digital persona with email and phone reservations:
```bash
./musu-nurikun forge [name] --project [project]
```

### 2. Autonomous Signup
Command the agent to register on a platform using an LLM to navigate the UI:
```bash
./musu-nurikun signup [platform] --id [ID]
```

### 3. Roaming (Warm-up)
Lurk and interact naturally to build account trust:
```bash
./musu-nurikun roam [url]
```

---

## 📂 Security & Privacy
All digital identity data, browser profiles, and session cookies are stored in the `projects/` directory. This data is strictly excluded from Git to prevent identity leaks.

---

## 🔗 The Ecosystem
- **musu-crawl-ai:** The "Brain" providing the knowledge to share.
- **musu-marketer:** The "Strategist" providing the content to speak.
