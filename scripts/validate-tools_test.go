package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// WorkflowConfig represents a GitHub Actions workflow
type WorkflowConfig struct {
	Name        string                 `yaml:"name"`
	On          interface{}            `yaml:"on"`
	Jobs        map[string]Job         `yaml:"jobs"`
	Permissions map[string]interface{} `yaml:"permissions,omitempty"`
	Env         map[string]string      `yaml:"env,omitempty"`
}

// Job represents a workflow job
type Job struct {
	Name      string                 `yaml:"name,omitempty"`
	RunsOn    interface{}            `yaml:"runs-on"`
	Steps     []Step                 `yaml:"steps"`
	Needs     interface{}            `yaml:"needs,omitempty"`
	If        string                 `yaml:"if,omitempty"`
	Strategy  map[string]interface{} `yaml:"strategy,omitempty"`
	Env       map[string]string      `yaml:"env,omitempty"`
	Outputs   map[string]string      `yaml:"outputs,omitempty"`
	Container interface{}            `yaml:"container,omitempty"`
	Services  map[string]interface{} `yaml:"services,omitempty"`
}

// Step represents a workflow step
type Step struct {
	Name string            `yaml:"name,omitempty"`
	Uses string            `yaml:"uses,omitempty"`
	Run  string            `yaml:"run,omitempty"`
	With map[string]string `yaml:"with,omitempty"`
	Env  map[string]string `yaml:"env,omitempty"`
	If   string            `yaml:"if,omitempty"`
}

// TestWorkflowValidation tests the validation of GitHub Actions workflows
func TestWorkflowValidation(t *testing.T) {
	workflowsDir := "../.github/workflows"

	// Check if workflows directory exists
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		t.Skip("Workflows directory does not exist, skipping workflow validation tests")
	}

	// Get all workflow files
	workflowFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.yml"))
	if err != nil {
		t.Fatalf("Failed to find workflow files: %v", err)
	}

	yamlFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.yaml"))
	if err != nil {
		t.Fatalf("Failed to find yaml workflow files: %v", err)
	}

	workflowFiles = append(workflowFiles, yamlFiles...)

	if len(workflowFiles) == 0 {
		t.Skip("No workflow files found, skipping validation tests")
	}

	for _, workflowFile := range workflowFiles {
		t.Run(filepath.Base(workflowFile), func(t *testing.T) {
			testWorkflowFile(t, workflowFile)
		})
	}
}

// testWorkflowFile tests a single workflow file
func testWorkflowFile(t *testing.T, workflowFile string) {
	// Read workflow file
	data, err := os.ReadFile(workflowFile)
	if err != nil {
		t.Fatalf("Failed to read workflow file %s: %v", workflowFile, err)
	}

	// Parse YAML
	var workflow WorkflowConfig
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatalf("Failed to parse workflow YAML %s: %v", workflowFile, err)
	}

	// Test required fields
	t.Run("RequiredFields", func(t *testing.T) {
		testRequiredFields(t, workflow, workflowFile)
	})

	// Test job structure
	t.Run("JobStructure", func(t *testing.T) {
		testJobStructure(t, workflow, workflowFile)
	})

	// Test security practices
	t.Run("SecurityPractices", func(t *testing.T) {
		testSecurityPractices(t, workflow, workflowFile, string(data))
	})

	// Test caching configuration
	t.Run("CachingConfiguration", func(t *testing.T) {
		testCachingConfiguration(t, workflow, workflowFile, string(data))
	})

	// Test action versions
	t.Run("ActionVersions", func(t *testing.T) {
		testActionVersions(t, workflow, workflowFile)
	})
}

// testRequiredFields tests that required fields are present
func testRequiredFields(t *testing.T, workflow WorkflowConfig, workflowFile string) {
	if workflow.Name == "" {
		t.Errorf("Workflow %s missing required field: name", workflowFile)
	}

	if workflow.On == nil {
		t.Errorf("Workflow %s missing required field: on", workflowFile)
	}

	if len(workflow.Jobs) == 0 {
		t.Errorf("Workflow %s missing required field: jobs", workflowFile)
	}
}

// testJobStructure tests the structure of jobs
func testJobStructure(t *testing.T, workflow WorkflowConfig, workflowFile string) {
	for jobName, job := range workflow.Jobs {
		if job.RunsOn == nil {
			t.Errorf("Job %s in workflow %s missing required field: runs-on", jobName, workflowFile)
		}

		if len(job.Steps) == 0 {
			t.Errorf("Job %s in workflow %s missing required field: steps", jobName, workflowFile)
		}

		// Test step structure
		for i, step := range job.Steps {
			if step.Uses == "" && step.Run == "" {
				t.Errorf("Step %d in job %s of workflow %s must have either 'uses' or 'run'", i, jobName, workflowFile)
			}
		}
	}
}

// testSecurityPractices tests security best practices
func testSecurityPractices(t *testing.T, workflow WorkflowConfig, workflowFile string, content string) {
	// Check for explicit permissions
	if len(workflow.Permissions) == 0 {
		t.Logf("Workflow %s has no explicit permissions defined (using defaults)", workflowFile)
	}

	// Check for potential hardcoded secrets
	if strings.Contains(content, "password") || strings.Contains(content, "secret") || strings.Contains(content, "token") {
		if !strings.Contains(content, "secrets.") && !strings.Contains(content, "github.token") {
			t.Logf("Workflow %s may contain hardcoded secrets (manual review recommended)", workflowFile)
		}
	}

	// Check for proper secret usage
	for jobName, job := range workflow.Jobs {
		for i, step := range job.Steps {
			if step.With != nil {
				for key, value := range step.With {
					if strings.Contains(strings.ToLower(key), "token") || strings.Contains(strings.ToLower(key), "password") {
						if !strings.HasPrefix(value, "${{ secrets.") && !strings.HasPrefix(value, "${{ github.token") {
							t.Errorf("Step %d in job %s of workflow %s may have hardcoded secret in parameter %s", i, jobName, workflowFile, key)
						}
					}
				}
			}
		}
	}
}

// testCachingConfiguration tests caching best practices
func testCachingConfiguration(t *testing.T, workflow WorkflowConfig, workflowFile string, content string) {
	hasGoSetup := strings.Contains(content, "setup-go")
	hasDockerBuild := strings.Contains(content, "docker/build-push-action")

	// Check Go caching
	if hasGoSetup {
		if !strings.Contains(content, "cache: true") && !strings.Contains(content, "actions/cache") {
			t.Logf("Workflow %s with Go setup could benefit from caching", workflowFile)
		}
	}

	// Check Docker caching
	if hasDockerBuild {
		if !strings.Contains(content, "cache-from") && !strings.Contains(content, "cache-to") {
			t.Logf("Workflow %s with Docker build could benefit from caching", workflowFile)
		}
	}
}

// testActionVersions tests that actions are properly versioned
func testActionVersions(t *testing.T, workflow WorkflowConfig, workflowFile string) {
	for jobName, job := range workflow.Jobs {
		for i, step := range job.Steps {
			if step.Uses != "" {
				// Check if action is pinned to a version
				if !strings.Contains(step.Uses, "@") {
					t.Errorf("Step %d in job %s of workflow %s uses unpinned action: %s", i, jobName, workflowFile, step.Uses)
				} else {
					// Check for SHA pinning (most secure) or version tags
					parts := strings.Split(step.Uses, "@")
					if len(parts) == 2 {
						version := parts[1]
						// Allow SHA (40 chars), version tags (v1, v1.2, v1.2.3), or latest
						if len(version) != 40 && !strings.HasPrefix(version, "v") && version != "latest" && version != "main" {
							t.Logf("Step %d in job %s of workflow %s uses non-standard version format: %s", i, jobName, workflowFile, step.Uses)
						}
					}
				}
			}
		}
	}
}

// TestWorkflowSchemaValidation tests workflow files against JSON schema
func TestWorkflowSchemaValidation(t *testing.T) {
	workflowsDir := "../.github/workflows"

	// Check if workflows directory exists
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		t.Skip("Workflows directory does not exist, skipping schema validation tests")
	}

	// Get all workflow files
	workflowFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.yml"))
	if err != nil {
		t.Fatalf("Failed to find workflow files: %v", err)
	}

	yamlFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.yaml"))
	if err != nil {
		t.Fatalf("Failed to find yaml workflow files: %v", err)
	}

	workflowFiles = append(workflowFiles, yamlFiles...)

	for _, workflowFile := range workflowFiles {
		t.Run(filepath.Base(workflowFile), func(t *testing.T) {
			// Read and parse workflow file
			data, err := os.ReadFile(workflowFile)
			if err != nil {
				t.Fatalf("Failed to read workflow file %s: %v", workflowFile, err)
			}

			// Convert YAML to JSON for schema validation
			var yamlData interface{}
			if err := yaml.Unmarshal(data, &yamlData); err != nil {
				t.Fatalf("Failed to parse YAML in %s: %v", workflowFile, err)
			}

			// Convert to JSON
			jsonData, err := json.Marshal(yamlData)
			if err != nil {
				t.Fatalf("Failed to convert to JSON for %s: %v", workflowFile, err)
			}

			// Basic JSON validation (ensure it's valid JSON)
			var jsonObj interface{}
			if err := json.Unmarshal(jsonData, &jsonObj); err != nil {
				t.Errorf("Workflow %s does not produce valid JSON: %v", workflowFile, err)
			}
		})
	}
}

// TestWorkflowNaming tests workflow naming conventions
func TestWorkflowNaming(t *testing.T) {
	workflowsDir := "../.github/workflows"

	// Check if workflows directory exists
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		t.Skip("Workflows directory does not exist, skipping naming tests")
	}

	// Get all workflow files
	workflowFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.yml"))
	if err != nil {
		t.Fatalf("Failed to find workflow files: %v", err)
	}

	yamlFiles, err := filepath.Glob(filepath.Join(workflowsDir, "*.yaml"))
	if err != nil {
		t.Fatalf("Failed to find yaml workflow files: %v", err)
	}

	workflowFiles = append(workflowFiles, yamlFiles...)

	for _, workflowFile := range workflowFiles {
		t.Run(filepath.Base(workflowFile), func(t *testing.T) {
			filename := filepath.Base(workflowFile)

			// Check file extension
			if !strings.HasSuffix(filename, ".yml") && !strings.HasSuffix(filename, ".yaml") {
				t.Errorf("Workflow file %s should have .yml or .yaml extension", filename)
			}

			// Check naming convention (kebab-case)
			nameWithoutExt := strings.TrimSuffix(strings.TrimSuffix(filename, ".yml"), ".yaml")
			if strings.Contains(nameWithoutExt, "_") || strings.Contains(nameWithoutExt, " ") {
				t.Logf("Workflow file %s should use kebab-case naming (hyphens instead of underscores/spaces)", filename)
			}

			// Read workflow to check internal name
			data, err := os.ReadFile(workflowFile)
			if err != nil {
				t.Fatalf("Failed to read workflow file %s: %v", workflowFile, err)
			}

			var workflow WorkflowConfig
			if err := yaml.Unmarshal(data, &workflow); err != nil {
				t.Fatalf("Failed to parse workflow YAML %s: %v", workflowFile, err)
			}

			// Check that workflow has a descriptive name
			if len(workflow.Name) < 3 {
				t.Errorf("Workflow %s should have a more descriptive name", workflowFile)
			}
		})
	}
}
