// Package main demonstrates the memory store stub integration with worker/planner components
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/agentflow/agentflow/internal/memory"
)

func main() {
	fmt.Println("AgentFlow Memory Store Demo")
	fmt.Println("===========================")

	// Check if experimental feature is enabled
	if os.Getenv("AF_MEMORY_ENABLED") != "true" {
		fmt.Println("Memory store is disabled by default (experimental feature)")
		fmt.Println("To enable: export AF_MEMORY_ENABLED=true")
		fmt.Println("Then run: go run cmd/memory-demo/main.go")
		return
	}

	ctx := context.Background()

	// Load configuration from environment
	config := memory.LoadConfigFromEnv()
	fmt.Printf("Memory store configuration: %+v\n", config)

	// Create memory container
	container, err := memory.NewContainer(config)
	if err != nil {
		log.Fatalf("Failed to create memory container: %v", err)
	}

	// Get memory store
	store, err := container.GetStore()
	if err != nil {
		log.Fatalf("Failed to get memory store: %v", err)
	}

	fmt.Println("\n1. Simulating Planner Operations")
	fmt.Println("---------------------------------")

	// Simulate planner creating a workflow plan
	workflowPlan := map[string]interface{}{
		"workflow_id": "demo_workflow_001",
		"type":        "customer_support",
		"plan": map[string]interface{}{
			"steps": []map[string]interface{}{
				{
					"id":     "classify",
					"agent":  "intent_classifier",
					"action": "classify_intent",
					"input":  map[string]interface{}{"message": "{{user_message}}"},
				},
				{
					"id":     "process",
					"agent":  "support_agent",
					"action": "handle_request",
					"input":  map[string]interface{}{"intent": "{{classify.output.intent}}"},
				},
				{
					"id":     "respond",
					"agent":  "response_generator",
					"action": "generate_response",
					"input":  map[string]interface{}{"resolution": "{{process.output.resolution}}"},
				},
			},
			"current_step": "classify",
			"status":       "active",
		},
		"created_at": time.Now().Format(time.RFC3339),
		"created_by": "planner:demo",
	}

	planKey := "plan:demo_workflow_001"
	if err := store.Save(ctx, planKey, workflowPlan); err != nil {
		log.Fatalf("Failed to save workflow plan: %v", err)
	}
	fmt.Printf("✓ Planner saved workflow plan: %s\n", planKey)

	fmt.Println("\n2. Simulating Worker Operations")
	fmt.Println("-------------------------------")

	// Simulate worker retrieving the plan
	query := memory.QueryRequest{Key: planKey}
	response, err := store.Query(ctx, query)
	if err != nil {
		log.Fatalf("Failed to query plan: %v", err)
	}

	if len(response.Entries) == 0 {
		log.Fatal("No plan found")
	}

	retrievedPlan := response.Entries[0]
	fmt.Printf("✓ Worker retrieved plan: %s\n", retrievedPlan.Key)

	// Simulate worker executing plan steps and storing intermediate results
	executionResults := []struct {
		key  string
		data interface{}
	}{
		{
			"execution:demo_workflow_001:step_classify",
			map[string]interface{}{
				"workflow_id": "demo_workflow_001",
				"step_id":     "classify",
				"agent":       "intent_classifier",
				"input":       map[string]interface{}{"message": "I need help with my order"},
				"output": map[string]interface{}{
					"intent":     "order_inquiry",
					"confidence": 0.92,
					"entities":   []string{"order"},
				},
				"status":       "completed",
				"duration_ms":  150,
				"completed_at": time.Now().Format(time.RFC3339),
			},
		},
		{
			"execution:demo_workflow_001:step_process",
			map[string]interface{}{
				"workflow_id": "demo_workflow_001",
				"step_id":     "process",
				"agent":       "support_agent",
				"input": map[string]interface{}{
					"intent":      "order_inquiry",
					"customer_id": "cust_12345",
				},
				"output": map[string]interface{}{
					"resolution": "Order status retrieved",
					"order_info": map[string]interface{}{
						"order_id": "ord_67890",
						"status":   "shipped",
						"tracking": "TRK123456789",
					},
				},
				"status":       "completed",
				"duration_ms":  320,
				"completed_at": time.Now().Format(time.RFC3339),
			},
		},
	}

	for _, result := range executionResults {
		if err := store.Save(ctx, result.key, result.data); err != nil {
			log.Fatalf("Failed to save execution result: %v", err)
		}
		fmt.Printf("✓ Worker saved execution result: %s\n", result.key)
	}

	fmt.Println("\n3. Simulating Agent Memory Operations")
	fmt.Println("------------------------------------")

	// Simulate agents storing learned patterns and context
	agentMemories := []struct {
		key  string
		data interface{}
	}{
		{
			"memory:intent_classifier:patterns",
			map[string]interface{}{
				"learned_patterns": map[string]interface{}{
					"order_inquiry": []string{
						"help with order", "order status", "where is my order",
						"track order", "order problem", "order issue",
					},
					"billing_question": []string{
						"billing issue", "charge problem", "refund request",
						"payment failed", "invoice question",
					},
					"technical_support": []string{
						"not working", "error message", "bug report",
						"technical issue", "system problem",
					},
				},
				"confidence_thresholds": map[string]float64{
					"order_inquiry":     0.85,
					"billing_question":  0.80,
					"technical_support": 0.90,
				},
				"last_updated": time.Now().Format(time.RFC3339),
			},
		},
		{
			"memory:support_agent:context",
			map[string]interface{}{
				"recent_interactions": []map[string]interface{}{
					{
						"customer_id":  "cust_12345",
						"intent":       "order_inquiry",
						"resolution":   "provided_tracking",
						"satisfaction": 4.5,
						"timestamp":    time.Now().Add(-10 * time.Minute).Format(time.RFC3339),
					},
				},
				"knowledge_base": map[string]interface{}{
					"common_solutions": map[string]string{
						"order_inquiry":     "Check order status and provide tracking information",
						"billing_question":  "Review billing details and process refunds if needed",
						"technical_support": "Escalate to technical team with detailed error information",
					},
				},
				"performance_metrics": map[string]interface{}{
					"avg_resolution_time": "4.2 minutes",
					"satisfaction_score":  4.3,
					"success_rate":        0.89,
				},
			},
		},
	}

	for _, agentMemory := range agentMemories {
		if err := store.Save(ctx, agentMemory.key, agentMemory.data); err != nil {
			log.Fatalf("Failed to save agent memory: %v", err)
		}
		fmt.Printf("✓ Agent saved memory: %s\n", agentMemory.key)
	}

	fmt.Println("\n4. Querying and Analysis")
	fmt.Println("------------------------")

	// Query all execution results for this workflow
	executionQuery := memory.QueryRequest{Prefix: "execution:demo_workflow_001:"}
	executionResponse, err := store.Query(ctx, executionQuery)
	if err != nil {
		log.Fatalf("Failed to query execution results: %v", err)
	}

	fmt.Printf("✓ Found %d execution results for workflow\n", len(executionResponse.Entries))
	for _, entry := range executionResponse.Entries {
		if data, ok := entry.Data.(map[string]interface{}); ok {
			if status, exists := data["status"]; exists {
				if duration, exists := data["duration_ms"]; exists {
					fmt.Printf("  - %s: %s (%v ms)\n", entry.Key, status, duration)
				}
			}
		}
	}

	// Query all agent memories
	memoryQuery := memory.QueryRequest{Prefix: "memory:"}
	memoryResponse, err := store.Query(ctx, memoryQuery)
	if err != nil {
		log.Fatalf("Failed to query agent memories: %v", err)
	}

	fmt.Printf("✓ Found %d agent memories\n", len(memoryResponse.Entries))
	for _, entry := range memoryResponse.Entries {
		fmt.Printf("  - %s (updated: %s)\n", entry.Key, entry.UpdatedAt.Format("15:04:05"))
	}

	fmt.Println("\n5. Summarization Demo (Q2.6 Placeholder)")
	fmt.Println("----------------------------------------")

	// Demonstrate summarization placeholder
	summarizeRequest := memory.SummarizeRequest{
		Context: "Customer support workflow execution summary",
		Data: []interface{}{
			workflowPlan,
			executionResults[0].data,
			executionResults[1].data,
		},
		Options: map[string]interface{}{
			"format":          "brief",
			"include_metrics": true,
			"focus":           "performance",
		},
	}

	summarizeResponse, err := store.Summarize(ctx, summarizeRequest)
	if err != nil {
		log.Fatalf("Failed to summarize: %v", err)
	}

	fmt.Printf("✓ Summary: %s\n", summarizeResponse.Summary)
	fmt.Printf("  Metadata: %+v\n", summarizeResponse.Metadata)

	fmt.Println("\n6. Performance Statistics")
	fmt.Println("-------------------------")

	// Get memory store statistics
	if inMemoryStore, ok := store.(*memory.InMemoryStore); ok {
		stats := inMemoryStore.GetStats()
		fmt.Printf("✓ Memory store statistics:\n")
		for key, value := range stats {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	// Perform health check
	if err := container.HealthCheck(ctx); err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		fmt.Println("✓ Health check passed")
	}

	fmt.Println("\n7. Data Export (JSON)")
	fmt.Println("--------------------")

	// Export all data as JSON for inspection
	allDataQuery := memory.QueryRequest{}
	allDataResponse, err := store.Query(ctx, allDataQuery)
	if err != nil {
		log.Fatalf("Failed to query all data: %v", err)
	}

	fmt.Printf("✓ Exporting %d entries:\n", len(allDataResponse.Entries))
	for i, entry := range allDataResponse.Entries {
		if i < 3 { // Show first 3 entries in detail
			entryJSON, _ := json.MarshalIndent(entry, "", "  ")
			fmt.Printf("\nEntry %d:\n%s\n", i+1, string(entryJSON))
		} else if i == 3 {
			fmt.Printf("... and %d more entries\n", len(allDataResponse.Entries)-3)
			break
		}
	}

	fmt.Println("\n=== Memory Store Demo Completed Successfully ===")
	fmt.Printf("Total entries stored: %d\n", len(allDataResponse.Entries))
	fmt.Println("Memory store is ready for worker/planner integration!")
}
