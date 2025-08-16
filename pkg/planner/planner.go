// Package planner provides planning interfaces for AgentFlow (FSM, BT, LLM)
package planner

import "context"

// Planner provides planning interface for workflow execution
type Planner interface {
	Plan(ctx context.Context, request PlanRequest) (Plan, error)
}

// PlanRequest represents a planning request
type PlanRequest struct {
	Goal        string
	Context     map[string]interface{}
	Constraints []Constraint
}

// Plan represents an execution plan
type Plan struct {
	ID    string
	Steps []Step
}

// Step represents a single step in a plan
type Step struct {
	ID        string
	AgentID   string
	Input     map[string]interface{}
	DependsOn []string
}

// Constraint represents a planning constraint
type Constraint struct {
	Type  string
	Value interface{}
}

// PlannerType represents different planner implementations
type PlannerType string

const (
	PlannerTypeFSM PlannerType = "fsm"
	PlannerTypeBT  PlannerType = "bt"
	PlannerTypeLLM PlannerType = "llm"
)
