package agent

// Asker is the minimal LLM chat surface the marketing crew depends on.
//
// The concrete *AgentClient (the OpenAI/Ollama HTTP client) satisfies it, and so
// does any injected generator — notably website-co's hardened subscription-CLI
// agent.Client, wrapped by the public marketer facade. Threading this interface
// through the Strategist/Copywriter/Critic lets the crew run on either transport
// without knowing which, so a single AI backend (the same gemini/claude CLI the
// content daemon already uses) can drive marketing without a separate LLM server.
type Asker interface {
	Ask(prompt string, jsonFormat bool) (string, error)
}
