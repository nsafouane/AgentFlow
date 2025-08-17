// Package messaging provides message bus abstractions for AgentFlow
package messaging

// Subject taxonomy constants for NATS message routing
const (
	// Workflow subjects
	SubjectWorkflowsPrefix = "workflows"
	SubjectWorkflowsIn     = "workflows.*.in"
	SubjectWorkflowsOut    = "workflows.*.out"

	// Agent subjects
	SubjectAgentsPrefix = "agents"
	SubjectAgentsIn     = "agents.*.in"
	SubjectAgentsOut    = "agents.*.out"

	// Tool subjects
	SubjectToolsPrefix = "tools"
	SubjectToolsCalls  = "tools.calls"
	SubjectToolsAudit  = "tools.audit"

	// System subjects
	SubjectSystemPrefix  = "system"
	SubjectSystemControl = "system.control"
	SubjectSystemHealth  = "system.health"
)

// SubjectBuilder provides utilities for building NATS subjects
type SubjectBuilder struct{}

// NewSubjectBuilder creates a new subject builder
func NewSubjectBuilder() *SubjectBuilder {
	return &SubjectBuilder{}
}

// WorkflowIn builds a workflow inbound subject
func (sb *SubjectBuilder) WorkflowIn(workflowID string) string {
	return "workflows." + workflowID + ".in"
}

// WorkflowOut builds a workflow outbound subject
func (sb *SubjectBuilder) WorkflowOut(workflowID string) string {
	return "workflows." + workflowID + ".out"
}

// AgentIn builds an agent inbound subject
func (sb *SubjectBuilder) AgentIn(agentID string) string {
	return "agents." + agentID + ".in"
}

// AgentOut builds an agent outbound subject
func (sb *SubjectBuilder) AgentOut(agentID string) string {
	return "agents." + agentID + ".out"
}
