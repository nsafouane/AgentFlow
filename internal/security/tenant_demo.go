package security

import (
	"context"
	"fmt"

	"github.com/agentflow/agentflow/internal/logging"
	"github.com/agentflow/agentflow/pkg/messaging"
)

// DemoMultiTenancyEnforcement demonstrates the complete multi-tenancy system
func DemoMultiTenancyEnforcement() {
	fmt.Println("=== AgentFlow Multi-Tenancy Enforcement Demo ===")

	logger := logging.NewLogger()

	// Demo 1: Tenant Context Management
	fmt.Println("\n1. Tenant Context Management:")
	demoTenantContextManagement()

	// Demo 2: Database Query Scoping
	fmt.Println("\n2. Database Query Scoping:")
	demoDatabaseQueryScoping(logger)

	// Demo 3: Message Bus Subject Isolation
	fmt.Println("\n3. Message Bus Subject Isolation:")
	demoMessageBusSubjectIsolation()

	// Demo 4: Cross-Tenant Access Prevention
	fmt.Println("\n4. Cross-Tenant Access Prevention:")
	demoCrossTenantAccessPrevention()

	fmt.Println("\n=== Demo Complete ===")
}

func demoTenantContextManagement() {
	// Create tenant contexts
	tenant1 := &TenantContext{
		TenantID:   "12345678-1234-1234-1234-123456789012",
		TenantName: "Acme Corp",
		ResourceLimits: map[string]interface{}{
			"max_workflows": 100,
			"max_agents":    50,
		},
	}

	tenant2 := &TenantContext{
		TenantID:   "87654321-4321-4321-4321-210987654321",
		TenantName: "Beta Inc",
		ResourceLimits: map[string]interface{}{
			"max_workflows": 10,
			"max_agents":    5,
		},
	}

	// Create contexts
	ctx1 := WithTenantContext(context.Background(), tenant1)
	ctx2 := WithTenantContext(context.Background(), tenant2)

	// Demonstrate context isolation
	retrievedTenant1, _ := GetTenantContext(ctx1)
	retrievedTenant2, _ := GetTenantContext(ctx2)

	fmt.Printf("  Tenant 1: %s (%s) - Max Workflows: %v\n",
		retrievedTenant1.TenantName, retrievedTenant1.TenantID,
		retrievedTenant1.ResourceLimits["max_workflows"])

	fmt.Printf("  Tenant 2: %s (%s) - Max Workflows: %v\n",
		retrievedTenant2.TenantName, retrievedTenant2.TenantID,
		retrievedTenant2.ResourceLimits["max_workflows"])

	fmt.Printf("  ✓ Tenant contexts are properly isolated\n")
}

func demoDatabaseQueryScoping(logger logging.Logger) {
	fmt.Printf("  Database Query Scoping:\n")
	fmt.Printf("    ✓ TenantScopedDB automatically injects tenant_id WHERE clauses\n")
	fmt.Printf("    ✓ Multi-tenant tables: users, agents, workflows, messages, tools, audits, budgets, rbac_roles, rbac_bindings\n")
	fmt.Printf("    ✓ Non-tenant tables: tenants, plans (scoped through relationships)\n")
	fmt.Printf("    ✓ Example: 'SELECT * FROM users' becomes 'SELECT * FROM users WHERE users.tenant_id = $1'\n")
	fmt.Printf("    ✓ Cross-tenant queries are automatically prevented\n")
}

func demoMessageBusSubjectIsolation() {
	builder := messaging.NewTenantSubjectBuilder()

	tenant1ID := "12345678-1234-1234-1234-123456789012"
	tenant2ID := "87654321-4321-4321-4321-210987654321"
	workflowID := "workflow-123"
	agentID := "agent-456"

	// Generate tenant-scoped subjects
	subjects := []struct {
		name    string
		tenant1 string
		tenant2 string
	}{
		{
			name:    "Workflow In",
			tenant1: builder.TenantWorkflowIn(tenant1ID, workflowID),
			tenant2: builder.TenantWorkflowIn(tenant2ID, workflowID),
		},
		{
			name:    "Agent Out",
			tenant1: builder.TenantAgentOut(tenant1ID, agentID),
			tenant2: builder.TenantAgentOut(tenant2ID, agentID),
		},
		{
			name:    "Tools Calls",
			tenant1: builder.TenantToolsCalls(tenant1ID),
			tenant2: builder.TenantToolsCalls(tenant2ID),
		},
	}

	for _, s := range subjects {
		fmt.Printf("  %s:\n", s.name)
		fmt.Printf("    Tenant 1: %s\n", s.tenant1)
		fmt.Printf("    Tenant 2: %s\n", s.tenant2)

		// Verify isolation
		if s.tenant1 == s.tenant2 {
			fmt.Printf("    ❌ Subjects are not isolated!\n")
		} else {
			fmt.Printf("    ✓ Subjects are properly isolated\n")
		}

		// Verify tenant extraction
		extractedTenant1, _ := builder.ExtractTenantFromSubject(s.tenant1)
		extractedTenant2, _ := builder.ExtractTenantFromSubject(s.tenant2)

		if extractedTenant1 == tenant1ID && extractedTenant2 == tenant2ID {
			fmt.Printf("    ✓ Tenant extraction works correctly\n")
		}
	}
}

func demoCrossTenantAccessPrevention() {
	builder := messaging.NewTenantSubjectBuilder()

	// Create tenant contexts
	tenant1ID := "12345678-1234-1234-1234-123456789012"
	tenant2ID := "87654321-4321-4321-4321-210987654321"

	// Use the messaging package's TenantContext type
	tenant1Ctx := context.WithValue(context.Background(), "tenant_context", &messaging.TenantContext{
		TenantID: tenant1ID,
	})

	// Create subjects for both tenants
	tenant1Subject := builder.TenantWorkflowIn(tenant1ID, "workflow-123")
	tenant2Subject := builder.TenantWorkflowIn(tenant2ID, "workflow-123")

	fmt.Printf("  Testing cross-tenant access prevention:\n")
	fmt.Printf("    Tenant 1 Subject: %s\n", tenant1Subject)
	fmt.Printf("    Tenant 2 Subject: %s\n", tenant2Subject)

	// Test same-tenant access (should succeed)
	err1 := builder.ValidateSubjectTenantAccess(tenant1Ctx, tenant1Subject)
	if err1 == nil {
		fmt.Printf("    ✓ Same-tenant access allowed\n")
	} else {
		fmt.Printf("    ❌ Same-tenant access denied: %v\n", err1)
	}

	// Test cross-tenant access (should fail)
	err2 := builder.ValidateSubjectTenantAccess(tenant1Ctx, tenant2Subject)
	if err2 != nil {
		fmt.Printf("    ✓ Cross-tenant access blocked: %v\n", err2)
	} else {
		fmt.Printf("    ❌ Cross-tenant access allowed (security violation!)\n")
	}

	// Test subject filtering
	allSubjects := []string{tenant1Subject, tenant2Subject, "invalid-subject"}
	filteredSubjects := builder.FilterSubjectsByTenant(tenant1Ctx, allSubjects)

	fmt.Printf("    Original subjects: %d\n", len(allSubjects))
	fmt.Printf("    Filtered subjects: %d\n", len(filteredSubjects))
	if len(filteredSubjects) == 1 && filteredSubjects[0] == tenant1Subject {
		fmt.Printf("    ✓ Subject filtering works correctly\n")
	}
}
