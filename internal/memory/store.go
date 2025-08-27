// Package memory provides in-memory storage implementation for AgentFlow
package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryStore provides the interface for memory operations
type MemoryStore interface {
	// Save stores a memory entry with the given key and data
	Save(ctx context.Context, key string, data interface{}) error

	// Query retrieves memory entries matching the given query
	Query(ctx context.Context, query QueryRequest) (QueryResponse, error)

	// Summarize provides a placeholder summarization method for Q2.6 compatibility
	Summarize(ctx context.Context, request SummarizeRequest) (SummarizeResponse, error)
}

// QueryRequest represents a memory query request
type QueryRequest struct {
	Key    string                 `json:"key,omitempty"`
	Prefix string                 `json:"prefix,omitempty"`
	Filter map[string]interface{} `json:"filter,omitempty"`
	Limit  int                    `json:"limit,omitempty"`
}

// QueryResponse represents a memory query response
type QueryResponse struct {
	Entries []MemoryEntry `json:"entries"`
	Total   int           `json:"total"`
}

// MemoryEntry represents a stored memory entry
type MemoryEntry struct {
	ID        string                 `json:"id"`
	Key       string                 `json:"key"`
	Data      interface{}            `json:"data"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// SummarizeRequest represents a summarization request (Q2.6 compatibility)
type SummarizeRequest struct {
	Context string                 `json:"context"`
	Data    []interface{}          `json:"data"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// SummarizeResponse represents a summarization response (Q2.6 compatibility)
type SummarizeResponse struct {
	Summary   string                 `json:"summary"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
}

// InMemoryStore implements MemoryStore using hash map storage with concurrent access safety
type InMemoryStore struct {
	mu      sync.RWMutex
	entries map[string]MemoryEntry
}

// NewInMemoryStore creates a new in-memory store instance
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		entries: make(map[string]MemoryEntry),
	}
}

// Save stores a memory entry with deterministic behavior
func (s *InMemoryStore) Save(ctx context.Context, key string, data interface{}) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	entryID := uuid.New().String()

	// Check if entry exists to determine if this is an update
	if existing, exists := s.entries[key]; exists {
		entryID = existing.ID
		now = existing.CreatedAt // Preserve creation time for updates
	}

	entry := MemoryEntry{
		ID:        entryID,
		Key:       key,
		Data:      data,
		Metadata:  make(map[string]interface{}),
		CreatedAt: now,
		UpdatedAt: time.Now(),
	}

	s.entries[key] = entry
	return nil
}

// Query retrieves memory entries with deterministic ordering
func (s *InMemoryStore) Query(ctx context.Context, query QueryRequest) (QueryResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matches []MemoryEntry

	// Handle exact key match
	if query.Key != "" {
		if entry, exists := s.entries[query.Key]; exists {
			matches = append(matches, entry)
		}
	} else {
		// Handle prefix matching or return all entries
		for key, entry := range s.entries {
			if query.Prefix == "" || matchesPrefix(key, query.Prefix) {
				matches = append(matches, entry)
			}
		}
	}

	// Apply limit if specified
	if query.Limit > 0 && len(matches) > query.Limit {
		matches = matches[:query.Limit]
	}

	return QueryResponse{
		Entries: matches,
		Total:   len(matches),
	}, nil
}

// Summarize returns a constant placeholder response for Q2.6 compatibility
func (s *InMemoryStore) Summarize(ctx context.Context, request SummarizeRequest) (SummarizeResponse, error) {
	// Placeholder implementation for Q2.6 compatibility
	// This will be replaced with actual summarization logic in Q2.6 Memory Subsystem MVP
	return SummarizeResponse{
		Summary: "Memory store stub - summarization not yet implemented (Q2.6 placeholder)",
		Metadata: map[string]interface{}{
			"stub_version":   "1.0.0",
			"context_length": len(request.Context),
			"data_count":     len(request.Data),
			"implementation": "placeholder",
		},
		Timestamp: time.Now(),
	}, nil
}

// GetStats returns statistics about the memory store
func (s *InMemoryStore) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"total_entries":   len(s.entries),
		"implementation":  "in_memory_hash_map",
		"concurrent_safe": true,
	}
}

// Clear removes all entries from the store (useful for testing)
func (s *InMemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = make(map[string]MemoryEntry)
}

// matchesPrefix checks if a key matches the given prefix
func matchesPrefix(key, prefix string) bool {
	if len(prefix) > len(key) {
		return false
	}
	return key[:len(prefix)] == prefix
}
