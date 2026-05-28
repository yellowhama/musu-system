// Package agent in marketer is now a thin facade over the shared
// github.com/yellowhama/musu-core/agent. The local AgentClient previously
// lived here in a triplicated copy (mirrored by crawl-ai and nurikun). The
// Phase B extraction moves the implementation into musu-core so all three
// CLIs share one client.
//
// The wrapper preserves the previous local signature so existing call sites
// (copywriter.go, critic.go, strategist.go) keep compiling unchanged.
package agent

import (
	coreagent "github.com/yellowhama/musu-core/agent"
)

// AgentClient aliases the shared client type. Callers that hold
// *agent.AgentClient pointers continue to work.
type AgentClient = coreagent.Client

// ExecutionTrace aliases the shared trace type.
type ExecutionTrace = coreagent.ExecutionTrace

// NewAgentClient builds a Client wired for marketer: telemetry under wikiDir
// so chat rounds for the Strategist/Copywriter/Critic crew are auditable
// alongside the crawler's traces.
func NewAgentClient(baseURL, model, wikiDir, project string) *AgentClient {
	return coreagent.New(baseURL, model,
		coreagent.WithTelemetry(wikiDir, project),
	)
}
