// Package agent provides agent interfaces and runtime for AgentFlow
package agent

import "context"

// Agent represents a single agent in the AgentFlow system
type Agent interface {
	ID() string
	Execute(ctx context.Context, input Input) (Output, error)
}

// Input represents input data for agent execution
type Input struct {
	Data map[string]interface{}
}

// Output represents output data from agent execution
type Output struct {
	Data   map[string]interface{}
	Status Status
}

// Status represents the execution status
type Status string

const (
	StatusSuccess Status = "success"
	StatusError   Status = "error"
	StatusPending Status = "pending"
)

// Registry provides agent registration and discovery
type Registry interface {
	Register(agent Agent) error
	Get(id string) (Agent, error)
	List() []Agent
}
