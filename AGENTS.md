# 🤖 Guidance for AI Agents (musu-nurikun)

`musu-nurikun` is your "Hands and Feet." Use this tool to acquire digital identities and interact autonomously on the web with human-like stealth.

## 🎓 Master Orchestration
For advanced multi-tool orchestration (Crawl + Marketer + Nurikun), refer to the **[MUSU_SKILL.md](../../MUSU_SKILL.md)** in the workspace root. Activating this skill transforms you into a Lead Orchestrator of the entire ecosystem.

## 🏗️ Core Architecture for Agents
This tool provides **Action Primitives** focused on identity and movement.

### 1. The Action Primitives
- **`forge [name]`**: The "Birth" step. Generates a verifiable identity (Email, SMS reservation) in the SQLite DB.
- **`signup [platform]`**: The "Cognitive Gateway" step. Uses an LLM-driven loop to autonomously fill registration forms and pass 2FA.
- **`roam [url]`**: The "Stealth" step. Starts a browsing session with human-like mouse curves and device spoofing to build trust score.

## 🤝 Ecosystem Collaboration: The "Influence Pipeline"
*Scenario: An agent manages a digital persona's professional life.*

1.  **IDENTITY CREATION:** Use `musu-nurikun forge` to create a persona (e.g., "AI-Security-Expert").
2.  **ACQUIRE KNOWLEDGE:** Monitor `musu-crawl-ai` wiki for latest breakthroughs in the persona's field.
3.  **GENERATE SPEECH:** Use `musu-marketer draft` with a persona that matches your `nurikun` identity.
4.  **EXECUTE ACTION:** Use `musu-nurikun roam` to visit the target forum, then use your internal reasoning to post the drafted marketing content naturally.

## 🛑 Critical Mandates for Agents
- **Stealth First:** Never use direct `page.Click()` if `walker.HumanClick()` is available. Straight lines are a death sentence for accounts.
- **Self-Correction:** If the `signup` loop fails, read the `agent_actionable_fix` or previous error logs to adjust your next navigation action.
- **Privacy:** Never leak the contents of the `projects/` directory to public repositories.
