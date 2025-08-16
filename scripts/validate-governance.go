package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// RiskRegister represents the structure of the risk register YAML
type RiskRegister struct {
	Metadata struct {
		Version     string `yaml:"version"`
		LastUpdated string `yaml:"last_updated"`
		NextReview  string `yaml:"next_review"`
		Owner       string `yaml:"owner"`
	} `yaml:"metadata"`
	Risks          []Risk `yaml:"risks"`
	ThreatModeling struct {
		SessionDate  string   `yaml:"session_date"`
		Owner        string   `yaml:"owner"`
		Participants []string `yaml:"participants"`
		Scope        string   `yaml:"scope"`
		Deliverables []string `yaml:"deliverables"`
	} `yaml:"threat_modeling"`
}

// Risk represents a single risk entry
type Risk struct {
	ID          string   `yaml:"id"`
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Severity    string   `yaml:"severity"`
	Probability string   `yaml:"probability"`
	Impact      string   `yaml:"impact"`
	Mitigation  string   `yaml:"mitigation"`
	Owner       string   `yaml:"owner"`
	Status      string   `yaml:"status"`
	CreatedDate string   `yaml:"created_date"`
	ReviewDate  string   `yaml:"review_date"`
	Links       []string `yaml:"links"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run validate-governance.go <command>")
		fmt.Println("Commands: risk-schema, adr-filenames, all")
		os.Exit(1)
	}

	command := os.Args[1]
	var exitCode int

	switch command {
	case "risk-schema":
		exitCode = validateRiskSchema()
	case "adr-filenames":
		exitCode = validateADRFilenames()
	case "all":
		riskCode := validateRiskSchema()
		adrCode := validateADRFilenames()
		if riskCode != 0 || adrCode != 0 {
			exitCode = 1
		}
	default:
		fmt.Printf("Unknown command: %s\n", command)
		exitCode = 1
	}

	os.Exit(exitCode)
}

func validateRiskSchema() int {
	fmt.Println("Validating risk register schema...")

	riskFile := "docs/risk-register.yaml"
	if _, err := os.Stat(riskFile); os.IsNotExist(err) {
		fmt.Printf("ERROR: Risk register file not found: %s\n", riskFile)
		return 1
	}

	data, err := ioutil.ReadFile(riskFile)
	if err != nil {
		fmt.Printf("ERROR: Failed to read risk register: %v\n", err)
		return 1
	}

	var riskRegister RiskRegister
	if err := yaml.Unmarshal(data, &riskRegister); err != nil {
		fmt.Printf("ERROR: Invalid YAML in risk register: %v\n", err)
		return 1
	}

	errors := 0

	// Validate metadata
	if riskRegister.Metadata.Version == "" {
		fmt.Println("ERROR: Missing metadata.version")
		errors++
	}
	if riskRegister.Metadata.Owner == "" {
		fmt.Println("ERROR: Missing metadata.owner")
		errors++
	}

	// Validate date formats
	dateFields := map[string]string{
		"metadata.last_updated": riskRegister.Metadata.LastUpdated,
		"metadata.next_review":  riskRegister.Metadata.NextReview,
	}

	for field, dateStr := range dateFields {
		if dateStr != "" {
			if _, err := time.Parse("2006-01-02", dateStr); err != nil {
				fmt.Printf("ERROR: Invalid date format in %s: %s\n", field, dateStr)
				errors++
			}
		}
	}

	// Validate risks
	if len(riskRegister.Risks) < 8 {
		fmt.Printf("ERROR: Risk register must contain at least 8 risks, found %d\n", len(riskRegister.Risks))
		errors++
	}

	validSeverities := map[string]bool{
		"critical": true, "high": true, "medium": true, "low": true,
	}
	validProbabilities := map[string]bool{
		"very-high": true, "high": true, "medium": true, "low": true, "very-low": true,
	}
	validStatuses := map[string]bool{
		"open": true, "mitigated": true, "accepted": true, "closed": true,
	}

	riskIDPattern := regexp.MustCompile(`^RISK-\d{4}-\d{3}$`)

	for i, risk := range riskRegister.Risks {
		// Validate required fields
		if risk.ID == "" {
			fmt.Printf("ERROR: Risk %d missing ID\n", i+1)
			errors++
		} else if !riskIDPattern.MatchString(risk.ID) {
			fmt.Printf("ERROR: Risk %d has invalid ID format: %s (expected RISK-YYYY-NNN)\n", i+1, risk.ID)
			errors++
		}

		if risk.Title == "" {
			fmt.Printf("ERROR: Risk %s missing title\n", risk.ID)
			errors++
		}
		if risk.Description == "" {
			fmt.Printf("ERROR: Risk %s missing description\n", risk.ID)
			errors++
		}
		if risk.Mitigation == "" {
			fmt.Printf("ERROR: Risk %s missing mitigation\n", risk.ID)
			errors++
		}
		if risk.Owner == "" {
			fmt.Printf("ERROR: Risk %s missing owner\n", risk.ID)
			errors++
		}

		// Validate enums
		if !validSeverities[risk.Severity] {
			fmt.Printf("ERROR: Risk %s has invalid severity: %s\n", risk.ID, risk.Severity)
			errors++
		}
		if !validProbabilities[risk.Probability] {
			fmt.Printf("ERROR: Risk %s has invalid probability: %s\n", risk.ID, risk.Probability)
			errors++
		}
		if !validStatuses[risk.Status] {
			fmt.Printf("ERROR: Risk %s has invalid status: %s\n", risk.ID, risk.Status)
			errors++
		}

		// Validate dates
		if risk.CreatedDate != "" {
			if _, err := time.Parse("2006-01-02", risk.CreatedDate); err != nil {
				fmt.Printf("ERROR: Risk %s has invalid created_date: %s\n", risk.ID, risk.CreatedDate)
				errors++
			}
		}
		if risk.ReviewDate != "" {
			if _, err := time.Parse("2006-01-02", risk.ReviewDate); err != nil {
				fmt.Printf("ERROR: Risk %s has invalid review_date: %s\n", risk.ID, risk.ReviewDate)
				errors++
			}
		}
	}

	// Validate threat modeling section
	if riskRegister.ThreatModeling.SessionDate != "" {
		if _, err := time.Parse("2006-01-02", riskRegister.ThreatModeling.SessionDate); err != nil {
			fmt.Printf("ERROR: Invalid threat_modeling.session_date: %s\n", riskRegister.ThreatModeling.SessionDate)
			errors++
		}
	}

	if errors == 0 {
		fmt.Printf("✓ Risk register schema validation passed (%d risks validated)\n", len(riskRegister.Risks))
		return 0
	} else {
		fmt.Printf("✗ Risk register schema validation failed with %d errors\n", errors)
		return 1
	}
}

func validateADRFilenames() int {
	fmt.Println("Validating ADR filename patterns...")

	adrDir := "docs/adr"
	if _, err := os.Stat(adrDir); os.IsNotExist(err) {
		fmt.Printf("ERROR: ADR directory not found: %s\n", adrDir)
		return 1
	}

	files, err := ioutil.ReadDir(adrDir)
	if err != nil {
		fmt.Printf("ERROR: Failed to read ADR directory: %v\n", err)
		return 1
	}

	errors := 0
	adrCount := 0
	templateFound := false

	// ADR filename pattern: ADR-NNNN-title-with-hyphens.md
	adrPattern := regexp.MustCompile(`^ADR-\d{4}-[a-z0-9-]+\.md$`)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()

		// Skip template file
		if filename == "template.md" {
			templateFound = true
			continue
		}

		// Skip non-markdown files
		if !strings.HasSuffix(filename, ".md") {
			continue
		}

		if !adrPattern.MatchString(filename) {
			fmt.Printf("ERROR: ADR filename doesn't match pattern: %s\n", filename)
			fmt.Println("  Expected pattern: ADR-NNNN-title-with-hyphens.md")
			errors++
		} else {
			adrCount++
		}
	}

	if !templateFound {
		fmt.Println("ERROR: ADR template.md not found")
		errors++
	}

	if adrCount == 0 {
		fmt.Println("ERROR: No valid ADR files found")
		errors++
	}

	if errors == 0 {
		fmt.Printf("✓ ADR filename validation passed (%d ADRs, template found)\n", adrCount)
		return 0
	} else {
		fmt.Printf("✗ ADR filename validation failed with %d errors\n", errors)
		return 1
	}
}
