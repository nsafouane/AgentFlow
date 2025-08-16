// Package memory provides memory store interfaces for AgentFlow
package memory

import "context"

// Store provides memory storage interface for agents
type Store interface {
	Set(ctx context.Context, key string, value interface{}) error
	Get(ctx context.Context, key string) (interface{}, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]string, error)
}

// Entry represents a memory entry
type Entry struct {
	Key       string
	Value     interface{}
	Timestamp int64
	TTL       int64
}

// VectorStore provides vector storage for embeddings
type VectorStore interface {
	Store(ctx context.Context, id string, vector []float64, metadata map[string]interface{}) error
	Search(ctx context.Context, vector []float64, limit int) ([]SearchResult, error)
}

// SearchResult represents a vector search result
type SearchResult struct {
	ID       string
	Score    float64
	Metadata map[string]interface{}
}
