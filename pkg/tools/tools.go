// Package tools provides tool interfaces and registry for AgentFlow
package tools

import "context"

// Tool represents a tool that can be executed by agents
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input ToolInput) (ToolOutput, error)
}

// ToolInput represents input for tool execution
type ToolInput struct {
	Parameters map[string]interface{}
}

// ToolOutput represents output from tool execution
type ToolOutput struct {
	Result map[string]interface{}
	Error  error
}

// Registry provides tool registration and discovery
type Registry interface {
	Register(tool Tool) error
	Get(name string) (Tool, error)
	List() []Tool
}

// ExecutionProfile defines security and resource constraints for tool execution
type ExecutionProfile struct {
	Timeout          int64
	MemoryLimit      int64
	NetworkAccess    bool
	FileSystemAccess bool
}
