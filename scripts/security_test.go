package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Mock report structures for testing
type GosecIssue struct {
	Severity   string `json:"severity"`
	Confidence string `json:"confidence"`
	RuleID     string `json:"rule_id"`
	Details    string `json:"details"`
	File       string `json:"file"`
	Code       string `json:"code"`
	Line       string `json:"line"`
}

type GosecReport struct {
	Issues []GosecIssue `json:"Issues"`
	Stats  struct {
		Files int `json:"files"`
		Lines int `json:"lines"`
		Nosec int `json:"nosec"`
		Found int `json:"found"`
	} `json:"Stats"`
}

type GovulncheckFinding struct {
	OSV     string `json:"osv"`
	Finding struct {
		Type    string `json:"type"`
		Trace   []any  `json:"trace"`
		FixedIn string `json:"fixed_in,omitempty"`
	} `json:"finding,omitempty"`
}

type GrypeMatch struct {
	Vulnerability struct {
		ID          string   `json:"id"`
		DataSource  string   `json:"dataSource"`
		Namespace   string   `json:"namespace"`
		Severity    string   `json:"severity"`
		URLs        []string `json:"urls"`
		Description string   `json:"description"`
		Cvss        []struct {
			Version string `json:"version"`
			Vector  string `json:"vector"`
			Metrics struct {
				BaseScore           float64 `json:"baseScore"`
				ExploitabilityScore float64 `json:"exploitabilityScore"`
				ImpactScore         float64 `json:"impactScore"`
			} `json:"metrics"`
		} `json:"cvss"`
		Fix struct {
			Versions []string `json:"versions"`
			State    string   `json:"state"`
		} `json:"fix"`
		Advisories []any `json:"advisories"`
	} `json:"vulnerability"`
	RelatedVulnerabilities []any `json:"relatedVulnerabilities"`
	MatchDetails           []struct {
		Type       string `json:"type"`
		Matcher    string `json:"matcher"`
		SearchedBy struct {
			Language  string `json:"language"`
			Namespace string `json:"namespace"`
		} `json:"searchedBy"`
		Found struct {
			VersionConstraint string `json:"versionConstraint"`
		} `json:"found"`
	} `json:"matchDetails"`
	Artifact struct {
		Name      string `json:"name"`
		Version   string `json:"version"`
		Type      string `json:"type"`
		Locations []struct {
			Path    string `json:"path"`
			LayerID string `json:"layerID,omitempty"`
		} `json:"locations"`
		Language  string   `json:"language"`
		Licenses  []string `json:"licenses"`
		CPEs      []string `json:"cpes"`
		PURL      string   `json:"purl"`
		Upstreams []any    `json:"upstreams"`
	} `json:"artifact"`
}

type GrypeReport struct {
	Matches    []GrypeMatch `json:"matches"`
	Source     any          `json:"source"`
	Distro     any          `json:"distro"`
	Descriptor any          `json:"descriptor"`
}

type GitleaksSecret struct {
	Description string   `json:"Description"`
	StartLine   int      `json:"StartLine"`
	EndLine     int      `json:"EndLine"`
	StartColumn int      `json:"StartColumn"`
	EndColumn   int      `json:"EndColumn"`
	Match       string   `json:"Match"`
	Secret      string   `json:"Secret"`
	File        string   `json:"File"`
	Commit      string   `json:"Commit"`
	Entropy     float64  `json:"Entropy"`
	Author      string   `json:"Author"`
	Email       string   `json:"Email"`
	Date        string   `json:"Date"`
	Message     string   `json:"Message"`
	Tags        []string `json:"Tags"`
	RuleID      string   `json:"RuleID"`
}

// SecurityThreshold represents severity thresholds
type SecurityThreshold int

const (
	Info SecurityThreshold = iota
	Low
	Medium
	High
	Critical
)

// String returns the string representation of the threshold
func (s SecurityThreshold) String() string {
	switch s {
	case Info:
		return "info"
	case Low:
		return "low"
	case Medium:
		return "medium"
	case High:
		return "high"
	case Critical:
		return "critical"
	default:
		return "unknown"
	}
}

// ParseSeverity converts string severity to SecurityThreshold
func ParseSeverity(severity string) SecurityThreshold {
	switch severity {
	case "CRITICAL", "critical", "Critical":
		return Critical
	case "HIGH", "high", "High":
		return High
	case "MEDIUM", "medium", "Medium":
		return Medium
	case "LOW", "low", "Low":
		return Low
	case "INFO", "info", "Info":
		return Info
	default:
		return Medium // Default to medium for unknown severities
	}
}

// MeetsThreshold checks if a severity meets the given threshold
func MeetsThreshold(severity, threshold string) bool {
	severityLevel := ParseSeverity(severity)
	thresholdLevel := ParseSeverity(threshold)
	return severityLevel >= thresholdLevel
}

// Test functions

func TestParseSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected SecurityThreshold
	}{
		{"CRITICAL", Critical},
		{"critical", Critical},
		{"Critical", Critical},
		{"HIGH", High},
		{"high", High},
		{"High", High},
		{"MEDIUM", Medium},
		{"medium", Medium},
		{"Medium", Medium},
		{"LOW", Low},
		{"low", Low},
		{"Low", Low},
		{"INFO", Info},
		{"info", Info},
		{"Info", Info},
		{"unknown", Medium},
		{"", Medium},
	}

	for _, test := range tests {
		result := ParseSeverity(test.input)
		if result != test.expected {
			t.Errorf("ParseSeverity(%q) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestMeetsThreshold(t *testing.T) {
	tests := []struct {
		severity  string
		threshold string
		expected  bool
	}{
		{"critical", "high", true},
		{"high", "high", true},
		{"medium", "high", false},
		{"low", "high", false},
		{"critical", "critical", true},
		{"high", "critical", false},
		{"medium", "medium", true},
		{"low", "medium", false},
		{"info", "info", true},
		{"unknown", "medium", true}, // unknown defaults to medium
	}

	for _, test := range tests {
		result := MeetsThreshold(test.severity, test.threshold)
		if result != test.expected {
			t.Errorf("MeetsThreshold(%q, %q) = %v, expected %v",
				test.severity, test.threshold, result, test.expected)
		}
	}
}

func TestParseGosecReport(t *testing.T) {
	// Create mock gosec report
	mockReport := GosecReport{
		Issues: []GosecIssue{
			{
				Severity:   "HIGH",
				Confidence: "HIGH",
				RuleID:     "G101",
				Details:    "Potential hardcoded credentials",
				File:       "main.go",
				Code:       "password := \"secret123\"",
				Line:       "10",
			},
			{
				Severity:   "MEDIUM",
				Confidence: "HIGH",
				RuleID:     "G102",
				Details:    "Bind to all interfaces",
				File:       "server.go",
				Code:       "http.ListenAndServe(\":8080\", nil)",
				Line:       "25",
			},
			{
				Severity:   "CRITICAL",
				Confidence: "HIGH",
				RuleID:     "G103",
				Details:    "Use of unsafe block",
				File:       "unsafe.go",
				Code:       "unsafe.Pointer(&data)",
				Line:       "15",
			},
		},
		Stats: struct {
			Files int `json:"files"`
			Lines int `json:"lines"`
			Nosec int `json:"nosec"`
			Found int `json:"found"`
		}{
			Files: 10,
			Lines: 1000,
			Nosec: 0,
			Found: 3,
		},
	}

	// Test parsing and threshold logic
	totalIssues := len(mockReport.Issues)
	if totalIssues != 3 {
		t.Errorf("Expected 3 total issues, got %d", totalIssues)
	}

	// Count high/critical issues
	highCriticalCount := 0
	for _, issue := range mockReport.Issues {
		if MeetsThreshold(issue.Severity, "high") {
			highCriticalCount++
		}
	}

	if highCriticalCount != 2 {
		t.Errorf("Expected 2 high/critical issues, got %d", highCriticalCount)
	}

	// Test threshold logic
	shouldFail := false
	threshold := "high"
	for _, issue := range mockReport.Issues {
		if MeetsThreshold(issue.Severity, threshold) {
			shouldFail = true
			break
		}
	}

	if !shouldFail {
		t.Error("Expected scan to fail with high threshold, but it passed")
	}
}

func TestParseGrypeReport(t *testing.T) {
	// Create mock grype report
	mockReport := GrypeReport{
		Matches: []GrypeMatch{
			{
				Vulnerability: struct {
					ID          string   `json:"id"`
					DataSource  string   `json:"dataSource"`
					Namespace   string   `json:"namespace"`
					Severity    string   `json:"severity"`
					URLs        []string `json:"urls"`
					Description string   `json:"description"`
					Cvss        []struct {
						Version string `json:"version"`
						Vector  string `json:"vector"`
						Metrics struct {
							BaseScore           float64 `json:"baseScore"`
							ExploitabilityScore float64 `json:"exploitabilityScore"`
							ImpactScore         float64 `json:"impactScore"`
						} `json:"metrics"`
					} `json:"cvss"`
					Fix struct {
						Versions []string `json:"versions"`
						State    string   `json:"state"`
					} `json:"fix"`
					Advisories []any `json:"advisories"`
				}{
					ID:          "CVE-2021-12345",
					Severity:    "High",
					Description: "High severity vulnerability",
				},
			},
			{
				Vulnerability: struct {
					ID          string   `json:"id"`
					DataSource  string   `json:"dataSource"`
					Namespace   string   `json:"namespace"`
					Severity    string   `json:"severity"`
					URLs        []string `json:"urls"`
					Description string   `json:"description"`
					Cvss        []struct {
						Version string `json:"version"`
						Vector  string `json:"vector"`
						Metrics struct {
							BaseScore           float64 `json:"baseScore"`
							ExploitabilityScore float64 `json:"exploitabilityScore"`
							ImpactScore         float64 `json:"impactScore"`
						} `json:"metrics"`
					} `json:"cvss"`
					Fix struct {
						Versions []string `json:"versions"`
						State    string   `json:"state"`
					} `json:"fix"`
					Advisories []any `json:"advisories"`
				}{
					ID:          "CVE-2021-67890",
					Severity:    "Medium",
					Description: "Medium severity vulnerability",
				},
			},
			{
				Vulnerability: struct {
					ID          string   `json:"id"`
					DataSource  string   `json:"dataSource"`
					Namespace   string   `json:"namespace"`
					Severity    string   `json:"severity"`
					URLs        []string `json:"urls"`
					Description string   `json:"description"`
					Cvss        []struct {
						Version string `json:"version"`
						Vector  string `json:"vector"`
						Metrics struct {
							BaseScore           float64 `json:"baseScore"`
							ExploitabilityScore float64 `json:"exploitabilityScore"`
							ImpactScore         float64 `json:"impactScore"`
						} `json:"metrics"`
					} `json:"cvss"`
					Fix struct {
						Versions []string `json:"versions"`
						State    string   `json:"state"`
					} `json:"fix"`
					Advisories []any `json:"advisories"`
				}{
					ID:          "CVE-2021-11111",
					Severity:    "Critical",
					Description: "Critical severity vulnerability",
				},
			},
		},
	}

	// Test parsing and threshold logic
	totalVulns := len(mockReport.Matches)
	if totalVulns != 3 {
		t.Errorf("Expected 3 total vulnerabilities, got %d", totalVulns)
	}

	// Count high/critical vulnerabilities
	highCriticalCount := 0
	for _, match := range mockReport.Matches {
		if MeetsThreshold(match.Vulnerability.Severity, "high") {
			highCriticalCount++
		}
	}

	if highCriticalCount != 2 {
		t.Errorf("Expected 2 high/critical vulnerabilities, got %d", highCriticalCount)
	}
}

func TestParseGitleaksReport(t *testing.T) {
	// Create mock gitleaks report
	mockSecrets := []GitleaksSecret{
		{
			Description: "AWS Access Key",
			StartLine:   10,
			EndLine:     10,
			Match:       "AKIAIOSFODNN7EXAMPLE",
			Secret:      "AKIAIOSFODNN7EXAMPLE",
			File:        "config.go",
			RuleID:      "aws-access-token",
		},
		{
			Description: "Generic API Key",
			StartLine:   25,
			EndLine:     25,
			Match:       "api_key_12345",
			Secret:      "api_key_12345",
			File:        "api.go",
			RuleID:      "generic-api-key",
		},
	}

	// Test parsing
	secretCount := len(mockSecrets)
	if secretCount != 2 {
		t.Errorf("Expected 2 secrets, got %d", secretCount)
	}

	// Gitleaks findings are always considered high severity
	// Any secrets found should fail the scan
	shouldFail := secretCount > 0
	if !shouldFail {
		t.Error("Expected scan to fail when secrets are found")
	}
}

func TestParseGovulncheckReport(t *testing.T) {
	// Create mock govulncheck findings (JSONL format)
	mockFindings := []GovulncheckFinding{
		{
			OSV: "GO-2021-0001",
			Finding: struct {
				Type    string `json:"type"`
				Trace   []any  `json:"trace"`
				FixedIn string `json:"fixed_in,omitempty"`
			}{
				Type:    "vulnerability",
				Trace:   []any{"main.go:10:5"},
				FixedIn: "v1.2.3",
			},
		},
		{
			OSV: "GO-2021-0002",
			Finding: struct {
				Type    string `json:"type"`
				Trace   []any  `json:"trace"`
				FixedIn string `json:"fixed_in,omitempty"`
			}{
				Type:    "vulnerability",
				Trace:   []any{"server.go:25:10"},
				FixedIn: "v2.1.0",
			},
		},
	}

	// Test parsing
	vulnCount := 0
	for _, finding := range mockFindings {
		if finding.Finding.Type == "vulnerability" {
			vulnCount++
		}
	}

	if vulnCount != 2 {
		t.Errorf("Expected 2 vulnerabilities, got %d", vulnCount)
	}

	// govulncheck findings are considered high severity by default
	shouldFail := vulnCount > 0
	if !shouldFail {
		t.Error("Expected scan to fail when vulnerabilities are found")
	}
}

func TestCreateMockReports(t *testing.T) {
	// Create temporary directory for test reports
	tempDir := t.TempDir()

	// Create mock gosec report
	gosecReport := GosecReport{
		Issues: []GosecIssue{
			{
				Severity:   "HIGH",
				Confidence: "HIGH",
				RuleID:     "G101",
				Details:    "Test vulnerability",
				File:       "test.go",
				Code:       "password := \"test\"",
				Line:       "1",
			},
		},
	}

	gosecData, err := json.MarshalIndent(gosecReport, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal gosec report: %v", err)
	}

	gosecFile := filepath.Join(tempDir, "gosec-report.json")
	err = os.WriteFile(gosecFile, gosecData, 0644)
	if err != nil {
		t.Fatalf("Failed to write gosec report: %v", err)
	}

	// Verify file was created and can be read
	if _, err := os.Stat(gosecFile); os.IsNotExist(err) {
		t.Error("Gosec report file was not created")
	}

	// Read and parse the file
	data, err := os.ReadFile(gosecFile)
	if err != nil {
		t.Fatalf("Failed to read gosec report: %v", err)
	}

	var parsedReport GosecReport
	err = json.Unmarshal(data, &parsedReport)
	if err != nil {
		t.Fatalf("Failed to parse gosec report: %v", err)
	}

	if len(parsedReport.Issues) != 1 {
		t.Errorf("Expected 1 issue in parsed report, got %d", len(parsedReport.Issues))
	}

	if parsedReport.Issues[0].Severity != "HIGH" {
		t.Errorf("Expected HIGH severity, got %s", parsedReport.Issues[0].Severity)
	}
}

func TestSecurityScanIntegration(t *testing.T) {
	// Integration test that simulates the full security scan workflow

	// Test threshold logic with various scenarios
	scenarios := []struct {
		name           string
		findings       map[string][]string // tool -> severities
		threshold      string
		expectedResult bool // true = should fail
	}{
		{
			name: "No vulnerabilities",
			findings: map[string][]string{
				"gosec": {},
				"grype": {},
			},
			threshold:      "high",
			expectedResult: false,
		},
		{
			name: "Only low severity",
			findings: map[string][]string{
				"gosec": {"low", "low"},
				"grype": {"low"},
			},
			threshold:      "high",
			expectedResult: false,
		},
		{
			name: "High severity present",
			findings: map[string][]string{
				"gosec": {"medium", "high"},
				"grype": {"low"},
			},
			threshold:      "high",
			expectedResult: true,
		},
		{
			name: "Critical severity present",
			findings: map[string][]string{
				"gosec": {"medium"},
				"grype": {"critical"},
			},
			threshold:      "high",
			expectedResult: true,
		},
		{
			name: "High threshold with critical findings",
			findings: map[string][]string{
				"gosec": {"critical", "high", "medium"},
			},
			threshold:      "critical",
			expectedResult: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			shouldFail := false

			// Check each tool's findings
			for _, severities := range scenario.findings {
				for _, severity := range severities {
					if MeetsThreshold(severity, scenario.threshold) {
						shouldFail = true
						break
					}
				}
				if shouldFail {
					break
				}
			}

			if shouldFail != scenario.expectedResult {
				t.Errorf("Scenario %s: expected shouldFail=%v, got %v",
					scenario.name, scenario.expectedResult, shouldFail)
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkParseSeverity(b *testing.B) {
	severities := []string{"critical", "high", "medium", "low", "info"}

	for i := 0; i < b.N; i++ {
		for _, severity := range severities {
			ParseSeverity(severity)
		}
	}
}

func BenchmarkMeetsThreshold(b *testing.B) {
	severities := []string{"critical", "high", "medium", "low", "info"}
	threshold := "high"

	for i := 0; i < b.N; i++ {
		for _, severity := range severities {
			MeetsThreshold(severity, threshold)
		}
	}
}
