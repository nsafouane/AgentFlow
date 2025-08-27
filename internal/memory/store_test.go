package memory

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestInMemoryStore_Save(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	tests := []struct {
		name    string
		key     string
		data    interface{}
		wantErr bool
	}{
		{
			name:    "valid save",
			key:     "test_key",
			data:    map[string]interface{}{"value": "test"},
			wantErr: false,
		},
		{
			name:    "empty key",
			key:     "",
			data:    "test_data",
			wantErr: true,
		},
		{
			name:    "nil data",
			key:     "test_key",
			data:    nil,
			wantErr: true,
		},
		{
			name:    "overwrite existing",
			key:     "test_key",
			data:    "new_value",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Save(ctx, tt.key, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInMemoryStore_Query(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Setup test data
	testData := map[string]interface{}{
		"user:1":   map[string]interface{}{"name": "Alice", "age": 30},
		"user:2":   map[string]interface{}{"name": "Bob", "age": 25},
		"config:1": map[string]interface{}{"setting": "value1"},
		"config:2": map[string]interface{}{"setting": "value2"},
	}

	for key, data := range testData {
		if err := store.Save(ctx, key, data); err != nil {
			t.Fatalf("Failed to save test data: %v", err)
		}
	}

	tests := []struct {
		name          string
		query         QueryRequest
		expectedCount int
		expectedKeys  []string
	}{
		{
			name:          "exact key match",
			query:         QueryRequest{Key: "user:1"},
			expectedCount: 1,
			expectedKeys:  []string{"user:1"},
		},
		{
			name:          "prefix match",
			query:         QueryRequest{Prefix: "user:"},
			expectedCount: 2,
			expectedKeys:  []string{"user:1", "user:2"},
		},
		{
			name:          "all entries",
			query:         QueryRequest{},
			expectedCount: 4,
			expectedKeys:  []string{"user:1", "user:2", "config:1", "config:2"},
		},
		{
			name:          "with limit",
			query:         QueryRequest{Prefix: "user:", Limit: 1},
			expectedCount: 1,
		},
		{
			name:          "non-existent key",
			query:         QueryRequest{Key: "non_existent"},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := store.Query(ctx, tt.query)
			if err != nil {
				t.Errorf("Query() error = %v", err)
				return
			}

			if response.Total != tt.expectedCount {
				t.Errorf("Query() total = %d, expected %d", response.Total, tt.expectedCount)
			}

			if len(response.Entries) != tt.expectedCount {
				t.Errorf("Query() entries count = %d, expected %d", len(response.Entries), tt.expectedCount)
			}

			// Check specific keys if provided
			if tt.expectedKeys != nil {
				foundKeys := make(map[string]bool)
				for _, entry := range response.Entries {
					foundKeys[entry.Key] = true
				}

				for _, expectedKey := range tt.expectedKeys {
					if !foundKeys[expectedKey] {
						t.Errorf("Query() missing expected key: %s", expectedKey)
					}
				}
			}
		})
	}
}

func TestInMemoryStore_Summarize(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	request := SummarizeRequest{
		Context: "test context",
		Data:    []interface{}{"item1", "item2", "item3"},
		Options: map[string]interface{}{"option1": "value1"},
	}

	response, err := store.Summarize(ctx, request)
	if err != nil {
		t.Errorf("Summarize() error = %v", err)
		return
	}

	// Verify placeholder response
	expectedSummary := "Memory store stub - summarization not yet implemented (Q2.6 placeholder)"
	if response.Summary != expectedSummary {
		t.Errorf("Summarize() summary = %s, expected %s", response.Summary, expectedSummary)
	}

	// Verify metadata
	if response.Metadata == nil {
		t.Error("Summarize() metadata is nil")
		return
	}

	if response.Metadata["stub_version"] != "1.0.0" {
		t.Errorf("Summarize() stub_version = %v, expected 1.0.0", response.Metadata["stub_version"])
	}

	if response.Metadata["context_length"] != len(request.Context) {
		t.Errorf("Summarize() context_length = %v, expected %d", response.Metadata["context_length"], len(request.Context))
	}

	if response.Metadata["data_count"] != len(request.Data) {
		t.Errorf("Summarize() data_count = %v, expected %d", response.Metadata["data_count"], len(request.Data))
	}

	if response.Metadata["implementation"] != "placeholder" {
		t.Errorf("Summarize() implementation = %v, expected placeholder", response.Metadata["implementation"])
	}

	// Verify timestamp is recent
	if time.Since(response.Timestamp) > time.Minute {
		t.Error("Summarize() timestamp is not recent")
	}
}

func TestInMemoryStore_DeterministicBehavior(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Test deterministic save/query behavior
	key := "deterministic_test"
	data := map[string]interface{}{"value": "test", "number": 42}

	// Save data
	err := store.Save(ctx, key, data)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Query multiple times and verify consistent results
	for i := 0; i < 5; i++ {
		response, err := store.Query(ctx, QueryRequest{Key: key})
		if err != nil {
			t.Errorf("Query() iteration %d error = %v", i, err)
			continue
		}

		if len(response.Entries) != 1 {
			t.Errorf("Query() iteration %d entries count = %d, expected 1", i, len(response.Entries))
			continue
		}

		entry := response.Entries[0]
		if entry.Key != key {
			t.Errorf("Query() iteration %d key = %s, expected %s", i, entry.Key, key)
		}

		// Verify data integrity
		entryData, ok := entry.Data.(map[string]interface{})
		if !ok {
			t.Errorf("Query() iteration %d data type assertion failed", i)
			continue
		}

		if entryData["value"] != "test" || entryData["number"] != 42 {
			t.Errorf("Query() iteration %d data integrity check failed: %v", i, entryData)
		}
	}
}

func TestInMemoryStore_ConcurrentAccess(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numOperations)

	// Test concurrent writes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("concurrent_%d_%d", goroutineID, j)
				data := map[string]interface{}{
					"goroutine": goroutineID,
					"operation": j,
					"timestamp": time.Now().UnixNano(),
				}
				if err := store.Save(ctx, key, data); err != nil {
					errors <- err
				}
			}
		}(i)
	}

	// Test concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				query := QueryRequest{Prefix: fmt.Sprintf("concurrent_%d_", goroutineID)}
				if _, err := store.Query(ctx, query); err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
	}

	// Verify final state
	response, err := store.Query(ctx, QueryRequest{Prefix: "concurrent_"})
	if err != nil {
		t.Errorf("Final query error: %v", err)
	}

	expectedCount := numGoroutines * numOperations
	if response.Total != expectedCount {
		t.Errorf("Final count = %d, expected %d", response.Total, expectedCount)
	}
}

func TestInMemoryStore_GetStats(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Initially empty
	stats := store.GetStats()
	if stats["total_entries"] != 0 {
		t.Errorf("Initial total_entries = %v, expected 0", stats["total_entries"])
	}

	// Add some entries
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("stats_test_%d", i)
		data := map[string]interface{}{"index": i}
		if err := store.Save(ctx, key, data); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// Check updated stats
	stats = store.GetStats()
	if stats["total_entries"] != 5 {
		t.Errorf("Updated total_entries = %v, expected 5", stats["total_entries"])
	}

	if stats["implementation"] != "in_memory_hash_map" {
		t.Errorf("Implementation = %v, expected in_memory_hash_map", stats["implementation"])
	}

	if stats["concurrent_safe"] != true {
		t.Errorf("Concurrent_safe = %v, expected true", stats["concurrent_safe"])
	}
}

func TestInMemoryStore_Clear(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Add some entries
	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("clear_test_%d", i)
		data := map[string]interface{}{"index": i}
		if err := store.Save(ctx, key, data); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// Verify entries exist
	response, err := store.Query(ctx, QueryRequest{})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if response.Total != 3 {
		t.Fatalf("Before clear total = %d, expected 3", response.Total)
	}

	// Clear store
	store.Clear()

	// Verify store is empty
	response, err = store.Query(ctx, QueryRequest{})
	if err != nil {
		t.Errorf("Query() after clear error = %v", err)
	}
	if response.Total != 0 {
		t.Errorf("After clear total = %d, expected 0", response.Total)
	}

	stats := store.GetStats()
	if stats["total_entries"] != 0 {
		t.Errorf("After clear total_entries = %v, expected 0", stats["total_entries"])
	}
}

func TestMatchesPrefix(t *testing.T) {
	tests := []struct {
		key      string
		prefix   string
		expected bool
	}{
		{"user:123", "user:", true},
		{"user:123", "user", true},
		{"user:123", "use", true},
		{"user:123", "user:123", true},
		{"user:123", "user:1234", false},
		{"user:123", "admin:", false},
		{"", "", true},
		{"test", "", true},
		{"", "test", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.key, tt.prefix), func(t *testing.T) {
			result := matchesPrefix(tt.key, tt.prefix)
			if result != tt.expected {
				t.Errorf("matchesPrefix(%s, %s) = %v, expected %v", tt.key, tt.prefix, result, tt.expected)
			}
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkInMemoryStore_Save(b *testing.B) {
	store := NewInMemoryStore()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench_save_%d", i)
		data := map[string]interface{}{"index": i, "timestamp": time.Now()}
		if err := store.Save(ctx, key, data); err != nil {
			b.Fatalf("Save() error = %v", err)
		}
	}
}

func BenchmarkInMemoryStore_Query(b *testing.B) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Setup test data
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("bench_query_%d", i)
		data := map[string]interface{}{"index": i}
		if err := store.Save(ctx, key, data); err != nil {
			b.Fatalf("Save() error = %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := QueryRequest{Prefix: "bench_query_"}
		if _, err := store.Query(ctx, query); err != nil {
			b.Fatalf("Query() error = %v", err)
		}
	}
}
