// Package agent in crawl-ai is now a thin facade over the shared
// github.com/yellowhama/musu-core/agent. The local AgentClient (chat + embed +
// vision) previously lived here in a triplicated copy (mirrored by marketer
// and nurikun). The Phase B extraction moves the implementation into musu-core
// so all three CLIs share one client.
//
// The wrapper preserves the previous local signature so existing call sites
// (compile.go, fetch.go, index.go, orchestrator.go, fetcher.go, planner.go,
// analyst.go, compiler.go) keep compiling unchanged.
package agent

import (
	coreagent "github.com/yellowhama/musu-core/agent"
)

// AgentClient aliases the shared client type. Callers that hold
// *agent.AgentClient pointers continue to work.
type AgentClient = coreagent.Client

// ExecutionTrace aliases the shared trace type.
type ExecutionTrace = coreagent.ExecutionTrace

// NewAgentClient builds a Client wired for crawl-ai: vision model for
// DescribeImage and telemetry under wikiDir.
func NewAgentClient(baseURL, model, visionModel, wikiDir, project string) *AgentClient {
	return coreagent.New(baseURL, model,
		coreagent.WithVisionModel(visionModel),
		coreagent.WithTelemetry(wikiDir, project),
	)
}
