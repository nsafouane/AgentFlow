package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"
)

// TestManualMemoryStoreIntegration demonstrates memory store functionality
// This test can be run manually to verify the memory store behavior
func TestManualMemoryStoreIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping manual integration test in short mode")
	}

	ctx := context.Background()

	// Create memory store with experimental feature enabled
	config := Config{
		Enabled:        true,
		Implementation: "in_memory",
		MaxEntries:     1000,
		Debug:          true,
	}

	container, err := NewContainer(config)
	if err != nil {
		t.Fatalf("Failed to create memory container: %v", err)
	}

	store, err := container.GetStore()
	if err != nil {
		t.Fatalf("Failed to get memory store: %v", err)
	}

	log.Println("=== Manual Memory Store Integration Test ===")

	// Test 1: Save sample plan data
	log.Println("Test 1: Saving sample plan data...")

	samplePlan := map[string]interface{}{
		"plan_id":  "plan_12345",
		"workflow": "customer_support",
		"steps": []map[string]interface{}{
			{
				"id":       "step_1",
				"agent":    "classifier",
				"action":   "classify_intent",
				"input":    map[string]interface{}{"message": "I need help with my order"},
				"output":   map[string]interface{}{"intent": "order_inquiry", "confidence": 0.95},
				"status":   "completed",
				"duration": "150ms",
			},
			{
				"id":       "step_2",
				"agent":    "order_agent",
				"action":   "lookup_order",
				"input":    map[string]interface{}{"customer_id": "cust_789", "intent": "order_inquiry"},
				"output":   map[string]interface{}{"order_status": "shipped", "tracking": "TRK123456"},
				"status":   "completed",
				"duration": "300ms",
			},
			{
				"id":       "step_3",
				"agent":    "response_generator",
				"action":   "generate_response",
				"input":    map[string]interface{}{"order_status": "shipped", "tracking": "TRK123456"},
				"output":   map[string]interface{}{"response": "Your order has shipped! Tracking: TRK123456"},
				"status":   "completed",
				"duration": "200ms",
			},
		},
		"metadata": map[string]interface{}{
			"tenant_id":    "tenant_abc",
			"user_id":      "user_456",
			"session_id":   "session_789",
			"total_cost":   0.0045,
			"total_tokens": 150,
			"created_at":   time.Now().Format(time.RFC3339),
			"completed_at": time.Now().Add(650 * time.Millisecond).Format(time.RFC3339),
		},
	}

	planKey := "plan:customer_support:plan_12345"
	if err := store.Save(ctx, planKey, samplePlan); err != nil {
		t.Fatalf("Failed to save plan: %v", err)
	}
	log.Printf("✓ Saved plan with key: %s", planKey)

	// Test 2: Save agent memory data
	log.Println("Test 2: Saving agent memory data...")

	agentMemories := []struct {
		key  string
		data interface{}
	}{
		{
			"memory:classifier:patterns",
			map[string]interface{}{
				"learned_patterns": []string{
					"order inquiry", "billing question", "technical support",
					"account access", "product information",
				},
				"confidence_thresholds": map[string]float64{
					"order_inquiry":       0.85,
					"billing_question":    0.80,
					"technical_support":   0.90,
					"account_access":      0.95,
					"product_information": 0.75,
				},
				"last_updated": time.Now().Format(time.RFC3339),
			},
		},
		{
			"memory:order_agent:cache",
			map[string]interface{}{
				"recent_lookups": map[string]interface{}{
					"cust_789": map[string]interface{}{
						"order_id":   "ord_12345",
						"status":     "shipped",
						"tracking":   "TRK123456",
						"cached_at":  time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
						"expires_at": time.Now().Add(55 * time.Minute).Format(time.RFC3339),
					},
				},
				"performance_metrics": map[string]interface{}{
					"avg_lookup_time": "280ms",
					"cache_hit_rate":  0.75,
					"total_lookups":   1247,
				},
			},
		},
		{
			"memory:response_generator:templates",
			map[string]interface{}{
				"templates": map[string]string{
					"order_shipped":    "Your order has shipped! Tracking: {{tracking_number}}",
					"order_pending":    "Your order is being processed and will ship soon.",
					"order_delivered":  "Your order was delivered on {{delivery_date}}.",
					"billing_resolved": "Your billing issue has been resolved. Reference: {{reference_id}}",
				},
				"usage_stats": map[string]int{
					"order_shipped":    45,
					"order_pending":    23,
					"order_delivered":  67,
					"billing_resolved": 12,
				},
			},
		},
	}

	for _, memory := range agentMemories {
		if err := store.Save(ctx, memory.key, memory.data); err != nil {
			t.Fatalf("Failed to save agent memory %s: %v", memory.key, err)
		}
		log.Printf("✓ Saved agent memory with key: %s", memory.key)
	}

	// Test 3: Query and retrieve data
	log.Println("Test 3: Querying stored data...")

	// Query specific plan
	planQuery := QueryRequest{Key: planKey}
	planResponse, err := store.Query(ctx, planQuery)
	if err != nil {
		t.Fatalf("Failed to query plan: %v", err)
	}

	if len(planResponse.Entries) != 1 {
		t.Fatalf("Expected 1 plan entry, got %d", len(planResponse.Entries))
	}

	log.Printf("✓ Retrieved plan: %s", planResponse.Entries[0].Key)

	// Pretty print the plan data
	planJSON, _ := json.MarshalIndent(planResponse.Entries[0].Data, "", "  ")
	log.Printf("Plan data:\n%s", string(planJSON))

	// Query all agent memories
	memoryQuery := QueryRequest{Prefix: "memory:"}
	memoryResponse, err := store.Query(ctx, memoryQuery)
	if err != nil {
		t.Fatalf("Failed to query memories: %v", err)
	}

	log.Printf("✓ Retrieved %d agent memories", len(memoryResponse.Entries))
	for _, entry := range memoryResponse.Entries {
		log.Printf("  - %s (created: %s)", entry.Key, entry.CreatedAt.Format(time.RFC3339))
	}

	// Test 4: Test summarization placeholder
	log.Println("Test 4: Testing summarization placeholder...")

	summarizeRequest := SummarizeRequest{
		Context: "Customer support workflow execution summary",
		Data: []interface{}{
			samplePlan,
			"Customer inquiry about order status",
			"Successfully resolved with tracking information",
		},
		Options: map[string]interface{}{
			"include_metrics": true,
			"format":          "brief",
		},
	}

	summarizeResponse, err := store.Summarize(ctx, summarizeRequest)
	if err != nil {
		t.Fatalf("Failed to summarize: %v", err)
	}

	log.Printf("✓ Summarization response: %s", summarizeResponse.Summary)
	log.Printf("  Metadata: %+v", summarizeResponse.Metadata)

	// Test 5: Performance and stats
	log.Println("Test 5: Performance and statistics...")

	stats := store.(*InMemoryStore).GetStats()
	log.Printf("✓ Memory store statistics:")
	for key, value := range stats {
		log.Printf("  %s: %v", key, value)
	}

	// Test 6: Concurrent access simulation
	log.Println("Test 6: Concurrent access simulation...")

	// Simulate multiple agents accessing memory concurrently
	done := make(chan bool, 3)

	// Agent 1: Classifier updating patterns
	go func() {
		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("memory:classifier:pattern_%d", i)
			data := map[string]interface{}{
				"pattern":    fmt.Sprintf("pattern_%d", i),
				"confidence": 0.8 + float64(i)*0.02,
				"timestamp":  time.Now().Format(time.RFC3339),
			}
			if err := store.Save(ctx, key, data); err != nil {
				log.Printf("Agent 1 save error: %v", err)
			}
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Agent 2: Order agent caching lookups
	go func() {
		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("memory:order_agent:lookup_%d", i)
			data := map[string]interface{}{
				"customer_id": fmt.Sprintf("cust_%d", i),
				"order_data": map[string]interface{}{
					"status": "active",
					"value":  100.0 + float64(i)*10.0,
				},
				"timestamp": time.Now().Format(time.RFC3339),
			}
			if err := store.Save(ctx, key, data); err != nil {
				log.Printf("Agent 2 save error: %v", err)
			}
			time.Sleep(15 * time.Millisecond)
		}
		done <- true
	}()

	// Agent 3: Response generator querying templates
	go func() {
		for i := 0; i < 5; i++ {
			query := QueryRequest{Prefix: "memory:response_generator:"}
			if _, err := store.Query(ctx, query); err != nil {
				log.Printf("Agent 3 query error: %v", err)
			}
			time.Sleep(8 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for all agents to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	log.Println("✓ Concurrent access simulation completed")

	// Final stats
	finalStats := store.(*InMemoryStore).GetStats()
	log.Printf("✓ Final memory store statistics:")
	for key, value := range finalStats {
		log.Printf("  %s: %v", key, value)
	}

	// Test 7: Health check
	log.Println("Test 7: Health check...")

	if err := container.HealthCheck(ctx); err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	log.Println("✓ Health check passed")

	log.Println("=== Manual Memory Store Integration Test Completed Successfully ===")
}

// TestManualExperimentalFeatureFlag demonstrates the experimental feature flag behavior
func TestManualExperimentalFeatureFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping manual experimental feature test in short mode")
	}

	log.Println("=== Manual Experimental Feature Flag Test ===")

	// Test 1: Default behavior (disabled)
	log.Println("Test 1: Default behavior (memory store disabled)...")

	defaultConfig := DefaultConfig()
	defaultContainer, err := NewContainer(defaultConfig)
	if err != nil {
		t.Fatalf("Failed to create default container: %v", err)
	}

	if defaultContainer.IsEnabled() {
		t.Error("Memory store should be disabled by default")
	}

	_, err = defaultContainer.GetStore()
	if err == nil {
		t.Error("GetStore() should fail when memory store is disabled")
	}

	log.Printf("✓ Default behavior: %s", err.Error())

	// Test 2: Experimental feature enabled
	log.Println("Test 2: Experimental feature enabled...")

	enabledConfig := Config{
		Enabled:        true,
		Implementation: "in_memory",
		MaxEntries:     1000,
		Debug:          true,
	}

	enabledContainer, err := NewContainer(enabledConfig)
	if err != nil {
		t.Fatalf("Failed to create enabled container: %v", err)
	}

	if !enabledContainer.IsEnabled() {
		t.Error("Memory store should be enabled when experimental flag is set")
	}

	store, err := enabledContainer.GetStore()
	if err != nil {
		t.Fatalf("GetStore() should succeed when memory store is enabled: %v", err)
	}

	log.Println("✓ Experimental feature enabled successfully")

	// Test 3: Basic functionality with experimental flag
	log.Println("Test 3: Basic functionality with experimental flag...")

	ctx := context.Background()
	testKey := "experimental:test"
	testData := map[string]interface{}{
		"message":      "Memory store is working with experimental flag",
		"timestamp":    time.Now().Format(time.RFC3339),
		"feature_flag": "AF_MEMORY_ENABLED=true",
	}

	if err := store.Save(ctx, testKey, testData); err != nil {
		t.Fatalf("Failed to save with experimental flag: %v", err)
	}

	query := QueryRequest{Key: testKey}
	response, err := store.Query(ctx, query)
	if err != nil {
		t.Fatalf("Failed to query with experimental flag: %v", err)
	}

	if len(response.Entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(response.Entries))
	}

	log.Printf("✓ Retrieved data with experimental flag: %s", response.Entries[0].Key)

	log.Println("=== Manual Experimental Feature Flag Test Completed Successfully ===")
}

// TestManualWorkerPlannerIntegration simulates integration with worker and planner components
func TestManualWorkerPlannerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping manual worker/planner integration test in short mode")
	}

	log.Println("=== Manual Worker/Planner Integration Test ===")

	ctx := context.Background()

	// Create memory store container (simulating dependency injection)
	config := Config{
		Enabled:        true,
		Implementation: "in_memory",
		MaxEntries:     1000,
		Debug:          true,
	}

	container, err := NewContainer(config)
	if err != nil {
		t.Fatalf("Failed to create memory container: %v", err)
	}

	store, err := container.GetStore()
	if err != nil {
		t.Fatalf("Failed to get memory store: %v", err)
	}

	// Simulate planner creating and storing a plan
	log.Println("Simulating planner creating and storing a plan...")

	plannerData := map[string]interface{}{
		"planner_type": "fsm",
		"workflow_id":  "wf_content_pipeline",
		"plan": map[string]interface{}{
			"states": []map[string]interface{}{
				{"id": "start", "type": "initial", "transitions": []string{"content_analysis"}},
				{"id": "content_analysis", "type": "agent", "agent": "content_analyzer", "transitions": []string{"content_generation", "error"}},
				{"id": "content_generation", "type": "agent", "agent": "content_generator", "transitions": []string{"quality_check"}},
				{"id": "quality_check", "type": "agent", "agent": "quality_checker", "transitions": []string{"publish", "revision"}},
				{"id": "revision", "type": "agent", "agent": "content_generator", "transitions": []string{"quality_check"}},
				{"id": "publish", "type": "final"},
				{"id": "error", "type": "error"},
			},
			"current_state": "content_analysis",
			"context": map[string]interface{}{
				"topic":         "AI in Healthcare",
				"target_length": 1500,
				"audience":      "technical",
				"deadline":      time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			},
		},
		"created_at": time.Now().Format(time.RFC3339),
		"created_by": "planner:fsm",
	}

	planKey := "plan:content_pipeline:wf_content_pipeline"
	if err := store.Save(ctx, planKey, plannerData); err != nil {
		t.Fatalf("Planner failed to save plan: %v", err)
	}
	log.Printf("✓ Planner stored plan: %s", planKey)

	// Simulate worker retrieving and executing plan
	log.Println("Simulating worker retrieving and executing plan...")

	query := QueryRequest{Key: planKey}
	response, err := store.Query(ctx, query)
	if err != nil {
		t.Fatalf("Worker failed to retrieve plan: %v", err)
	}

	if len(response.Entries) != 1 {
		t.Fatalf("Worker expected 1 plan, got %d", len(response.Entries))
	}

	retrievedPlan := response.Entries[0]
	log.Printf("✓ Worker retrieved plan: %s (created: %s)", retrievedPlan.Key, retrievedPlan.CreatedAt.Format(time.RFC3339))

	// Simulate worker updating plan execution state
	log.Println("Simulating worker updating plan execution state...")

	// Update plan with execution progress
	planData := retrievedPlan.Data.(map[string]interface{})
	planMap := planData["plan"].(map[string]interface{})
	planMap["current_state"] = "content_generation"
	planMap["execution_log"] = []map[string]interface{}{
		{
			"state":        "content_analysis",
			"agent":        "content_analyzer",
			"started_at":   time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
			"completed_at": time.Now().Add(-2 * time.Minute).Format(time.RFC3339),
			"result": map[string]interface{}{
				"analysis":           "Topic requires technical depth with practical examples",
				"keywords":           []string{"AI", "healthcare", "machine learning", "diagnosis", "treatment"},
				"estimated_sections": 5,
			},
		},
	}
	planData["updated_at"] = time.Now().Format(time.RFC3339)
	planData["updated_by"] = "worker:agent_executor"

	if err := store.Save(ctx, planKey, planData); err != nil {
		t.Fatalf("Worker failed to update plan: %v", err)
	}
	log.Printf("✓ Worker updated plan execution state")

	// Simulate agent storing intermediate results in memory
	log.Println("Simulating agents storing intermediate results...")

	agentResults := []struct {
		key  string
		data interface{}
	}{
		{
			"memory:content_analyzer:analysis_wf_content_pipeline",
			map[string]interface{}{
				"workflow_id": "wf_content_pipeline",
				"analysis_result": map[string]interface{}{
					"complexity_score": 0.75,
					"technical_level":  "intermediate",
					"required_research": []string{
						"Recent AI healthcare applications",
						"FDA regulations for AI medical devices",
						"Case studies from major hospitals",
					},
					"content_structure": map[string]interface{}{
						"introduction":         "Overview of AI in healthcare landscape",
						"current_applications": "Diagnostic imaging, drug discovery, patient monitoring",
						"challenges":           "Regulatory compliance, data privacy, integration",
						"future_trends":        "Personalized medicine, predictive analytics",
						"conclusion":           "Implementation recommendations",
					},
				},
				"processing_time": "180ms",
				"confidence":      0.92,
				"timestamp":       time.Now().Format(time.RFC3339),
			},
		},
		{
			"memory:content_generator:draft_wf_content_pipeline",
			map[string]interface{}{
				"workflow_id": "wf_content_pipeline",
				"draft_sections": map[string]interface{}{
					"introduction": map[string]interface{}{
						"content":    "Artificial Intelligence is revolutionizing healthcare delivery...",
						"word_count": 245,
						"status":     "completed",
					},
					"current_applications": map[string]interface{}{
						"content":    "AI applications in healthcare span multiple domains...",
						"word_count": 387,
						"status":     "in_progress",
					},
				},
				"total_word_count":      632,
				"target_word_count":     1500,
				"completion_percentage": 0.42,
				"timestamp":             time.Now().Format(time.RFC3339),
			},
		},
	}

	for _, result := range agentResults {
		if err := store.Save(ctx, result.key, result.data); err != nil {
			t.Fatalf("Agent failed to save result %s: %v", result.key, err)
		}
		log.Printf("✓ Agent stored result: %s", result.key)
	}

	// Simulate planner querying agent memories for replanning
	log.Println("Simulating planner querying agent memories for replanning...")

	memoryQuery := QueryRequest{Prefix: "memory:"}
	memoryResponse, err := store.Query(ctx, memoryQuery)
	if err != nil {
		t.Fatalf("Planner failed to query memories: %v", err)
	}

	log.Printf("✓ Planner retrieved %d agent memories for replanning", len(memoryResponse.Entries))

	// Simulate planner using memories to adjust plan
	for _, memory := range memoryResponse.Entries {
		log.Printf("  - Processing memory: %s", memory.Key)

		// Simulate planner analyzing memory data for replanning decisions
		if memoryData, ok := memory.Data.(map[string]interface{}); ok {
			if workflowID, exists := memoryData["workflow_id"]; exists && workflowID == "wf_content_pipeline" {
				log.Printf("    Found relevant memory for workflow: %s", workflowID)

				// Simulate planner decision based on memory content
				if memory.Key == "memory:content_generator:draft_wf_content_pipeline" {
					if completion, ok := memoryData["completion_percentage"].(float64); ok && completion < 0.5 {
						log.Printf("    Planner decision: Content generation behind schedule (%.0f%% complete)", completion*100)
						log.Printf("    Planner action: Consider parallel content generation for remaining sections")
					}
				}
			}
		}
	}

	// Final statistics
	log.Println("Integration test statistics...")

	stats := store.(*InMemoryStore).GetStats()
	log.Printf("✓ Final memory store statistics:")
	for key, value := range stats {
		log.Printf("  %s: %v", key, value)
	}

	log.Println("=== Manual Worker/Planner Integration Test Completed Successfully ===")
}
