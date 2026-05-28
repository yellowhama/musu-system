// Package agent in nurikun is now a thin facade over the shared
// github.com/yellowhama/musu-system/core/agent. The local AgentClient type, the
// telemetry logTrace, and the OpenAI wire types previously lived here in a
// triplicated copy (mirrored by crawl-ai and marketer). The Phase B extraction
// moves the implementation into musu-core so all three CLIs share one client.
//
// The wrapper preserves the previous local signatures so existing call sites
// (watch.go, responder.go, triage.go) keep compiling unchanged.
package agent

import (
	coreagent "github.com/yellowhama/musu-system/core/agent"
)

// AgentClient aliases the shared client type. Callers that hold
// *agent.AgentClient pointers continue to work.
type AgentClient = coreagent.Client

// ExecutionTrace aliases the shared trace type for any in-tree consumer.
type ExecutionTrace = coreagent.ExecutionTrace

// NewAgentClient builds a Client wired for nurikun: telemetry under wikiDir,
// and trace records tagged with the "nurikun_agent" role label.
func NewAgentClient(baseURL, model, wikiDir, project string) *AgentClient {
	return coreagent.New(baseURL, model,
		coreagent.WithTelemetry(wikiDir, project),
		coreagent.WithRole("nurikun_agent"),
	)
}
