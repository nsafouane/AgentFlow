package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// SPDXDocument represents the structure of an SPDX SBOM
type SPDXDocument struct {
	SPDXVersion       string         `json:"spdxVersion"`
	DataLicense       string         `json:"dataLicense"`
	SPDXID            string         `json:"SPDXID"`
	Name              string         `json:"name"`
	DocumentNamespace string         `json:"documentNamespace"`
	CreationInfo      CreationInfo   `json:"creationInfo"`
	Packages          []Package      `json:"packages"`
	Files             []File         `json:"files,omitempty"`
	Relationships     []Relationship `json:"relationships,omitempty"`
}

type CreationInfo struct {
	LicenseListVersion string   `json:"licenseListVersion"`
	Creators           []string `json:"creators"`
	Created            string   `json:"created"`
}

type Package struct {
	Name             string        `json:"name"`
	SPDXID           string        `json:"SPDXID"`
	VersionInfo      string        `json:"versionInfo,omitempty"`
	Supplier         string        `json:"supplier,omitempty"`
	DownloadLocation string        `json:"downloadLocation"`
	FilesAnalyzed    bool          `json:"filesAnalyzed"`
	SourceInfo       string        `json:"sourceInfo,omitempty"`
	LicenseConcluded string        `json:"licenseConcluded"`
	LicenseDeclared  string        `json:"licenseDeclared"`
	CopyrightText    string        `json:"copyrightText"`
	ExternalRefs     []ExternalRef `json:"externalRefs,omitempty"`
}

type ExternalRef struct {
	ReferenceCategory string `json:"referenceCategory"`
	ReferenceType     string `json:"referenceType"`
	ReferenceLocator  string `json:"referenceLocator"`
}

type File struct {
	FileName           string     `json:"fileName"`
	SPDXID             string     `json:"SPDXID"`
	FileTypes          []string   `json:"fileTypes,omitempty"`
	Checksums          []Checksum `json:"checksums,omitempty"`
	LicenseConcluded   string     `json:"licenseConcluded"`
	LicenseInfoInFiles []string   `json:"licenseInfoInFiles"`
	CopyrightText      string     `json:"copyrightText"`
	Comment            string     `json:"comment,omitempty"`
}

type Checksum struct {
	Algorithm     string `json:"algorithm"`
	ChecksumValue string `json:"checksumValue"`
}

type Relationship struct {
	SPDXElementID      string `json:"spdxElementId"`
	RelatedSPDXElement string `json:"relatedSpdxElement"`
	RelationshipType   string `json:"relationshipType"`
	Comment            string `json:"comment,omitempty"`
}

// CycloneDXDocument represents the structure of a CycloneDX SBOM
type CycloneDXDocument struct {
	BOMFormat    string      `json:"bomFormat"`
	SpecVersion  string      `json:"specVersion"`
	SerialNumber string      `json:"serialNumber"`
	Version      int         `json:"version"`
	Metadata     Metadata    `json:"metadata"`
	Components   []Component `json:"components"`
}

type Metadata struct {
	Timestamp string    `json:"timestamp"`
	Tools     []Tool    `json:"tools"`
	Component Component `json:"component"`
}

type Tool struct {
	Vendor  string `json:"vendor"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Component struct {
	Type         string    `json:"type"`
	BOMRef       string    `json:"bom-ref,omitempty"`
	Name         string    `json:"name"`
	Version      string    `json:"version,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	Hashes       []Hash    `json:"hashes,omitempty"`
	Licenses     []License `json:"licenses,omitempty"`
	PURL         string    `json:"purl,omitempty"`
	ExternalRefs []ExtRef  `json:"externalReferences,omitempty"`
}

type Hash struct {
	Algorithm string `json:"alg"`
	Content   string `json:"content"`
}

type License struct {
	License LicenseChoice `json:"license"`
}

type LicenseChoice struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type ExtRef struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// Test helper functions
func createTempSBOM(t *testing.T, format string, valid bool) string {
	tempDir := t.TempDir()
	var filename string
	var content []byte
	var err error

	switch format {
	case "spdx":
		filename = filepath.Join(tempDir, "test.spdx.json")
		if valid {
			sbom := SPDXDocument{
				SPDXVersion:       "SPDX-2.3",
				DataLicense:       "CC0-1.0",
				SPDXID:            "SPDXRef-DOCUMENT",
				Name:              "test-sbom",
				DocumentNamespace: "https://example.com/test",
				CreationInfo: CreationInfo{
					LicenseListVersion: "3.27",
					Creators:           []string{"Tool: test"},
					Created:            "2024-01-01T00:00:00Z",
				},
				Packages: []Package{
					{
						Name:             "test-package",
						SPDXID:           "SPDXRef-Package-test",
						VersionInfo:      "1.0.0",
						DownloadLocation: "NOASSERTION",
						FilesAnalyzed:    false,
						LicenseConcluded: "NOASSERTION",
						LicenseDeclared:  "MIT",
						CopyrightText:    "NOASSERTION",
					},
				},
			}
			content, err = json.Marshal(sbom)
		} else {
			content = []byte(`{"invalid": "json"`)
		}

	case "cyclonedx":
		filename = filepath.Join(tempDir, "test.cyclonedx.json")
		if valid {
			sbom := CycloneDXDocument{
				BOMFormat:    "CycloneDX",
				SpecVersion:  "1.4",
				SerialNumber: "urn:uuid:test",
				Version:      1,
				Metadata: Metadata{
					Timestamp: "2024-01-01T00:00:00Z",
					Tools: []Tool{
						{
							Vendor:  "test",
							Name:    "test-tool",
							Version: "1.0.0",
						},
					},
				},
				Components: []Component{
					{
						Type:    "library",
						BOMRef:  "test-component",
						Name:    "test-component",
						Version: "1.0.0",
					},
				},
			}
			content, err = json.Marshal(sbom)
		} else {
			content = []byte(`{"invalid": "json"`)
		}

	default:
		t.Fatalf("Unknown format: %s", format)
	}

	if err != nil {
		t.Fatalf("Failed to marshal SBOM: %v", err)
	}

	if err := os.WriteFile(filename, content, 0644); err != nil {
		t.Fatalf("Failed to write SBOM file: %v", err)
	}

	return filename
}

func TestSPDXSBOMValidation(t *testing.T) {
	tests := []struct {
		name        string
		valid       bool
		expectError bool
	}{
		{
			name:        "Valid SPDX SBOM",
			valid:       true,
			expectError: false,
		},
		{
			name:        "Invalid SPDX SBOM",
			valid:       false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sbomFile := createTempSBOM(t, "spdx", tt.valid)

			// Test SBOM validation by reading and parsing
			content, err := os.ReadFile(sbomFile)
			if err != nil {
				t.Fatalf("Failed to read SBOM file: %v", err)
			}

			var sbom SPDXDocument
			err = json.Unmarshal(content, &sbom)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error parsing invalid SBOM, but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Failed to parse valid SBOM: %v", err)
			}

			// Validate required fields
			if sbom.SPDXVersion == "" {
				t.Error("SPDX version is missing")
			}

			if len(sbom.Packages) == 0 {
				t.Error("SBOM contains no packages")
			}

			if sbom.SPDXID == "" {
				t.Error("SPDX ID is missing")
			}

			if sbom.DocumentNamespace == "" {
				t.Error("Document namespace is missing")
			}
		})
	}
}

func TestCycloneDXSBOMValidation(t *testing.T) {
	tests := []struct {
		name        string
		valid       bool
		expectError bool
	}{
		{
			name:        "Valid CycloneDX SBOM",
			valid:       true,
			expectError: false,
		},
		{
			name:        "Invalid CycloneDX SBOM",
			valid:       false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sbomFile := createTempSBOM(t, "cyclonedx", tt.valid)

			// Test SBOM validation by reading and parsing
			content, err := os.ReadFile(sbomFile)
			if err != nil {
				t.Fatalf("Failed to read SBOM file: %v", err)
			}

			var sbom CycloneDXDocument
			err = json.Unmarshal(content, &sbom)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error parsing invalid SBOM, but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Failed to parse valid SBOM: %v", err)
			}

			// Validate required fields
			if sbom.SpecVersion == "" {
				t.Error("Spec version is missing")
			}

			if len(sbom.Components) == 0 {
				t.Error("SBOM contains no components")
			}

			if sbom.BOMFormat == "" {
				t.Error("BOM format is missing")
			}

			if sbom.SerialNumber == "" {
				t.Error("Serial number is missing")
			}
		})
	}
}

func TestSBOMGenerationWithSyft(t *testing.T) {
	// Check if syft is available
	if _, err := exec.LookPath("syft"); err != nil {
		t.Skip("syft not available, skipping SBOM generation test")
	}

	// Create a temporary directory with a simple Go module
	tempDir := t.TempDir()

	// Create a simple go.mod file
	goMod := `module test-module

go 1.22

require (
	github.com/stretchr/testify v1.8.4
)
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create a simple main.go file
	mainGo := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}

	// Generate SPDX SBOM
	spdxFile := filepath.Join(tempDir, "sbom.spdx.json")
	cmd := exec.Command("syft", tempDir, "-o", fmt.Sprintf("spdx-json=%s", spdxFile), "--quiet")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to generate SPDX SBOM: %v", err)
	}

	// Validate generated SPDX SBOM
	content, err := os.ReadFile(spdxFile)
	if err != nil {
		t.Fatalf("Failed to read generated SPDX SBOM: %v", err)
	}

	var spdxSbom SPDXDocument
	if err := json.Unmarshal(content, &spdxSbom); err != nil {
		t.Fatalf("Failed to parse generated SPDX SBOM: %v", err)
	}

	if spdxSbom.SPDXVersion == "" {
		t.Error("Generated SPDX SBOM missing version")
	}

	if len(spdxSbom.Packages) == 0 {
		t.Error("Generated SPDX SBOM contains no packages")
	}

	// Generate CycloneDX SBOM
	cyclonedxFile := filepath.Join(tempDir, "sbom.cyclonedx.json")
	cmd = exec.Command("syft", tempDir, "-o", fmt.Sprintf("cyclonedx-json=%s", cyclonedxFile), "--quiet")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to generate CycloneDX SBOM: %v", err)
	}

	// Validate generated CycloneDX SBOM
	content, err = os.ReadFile(cyclonedxFile)
	if err != nil {
		t.Fatalf("Failed to read generated CycloneDX SBOM: %v", err)
	}

	var cyclonedxSbom CycloneDXDocument
	if err := json.Unmarshal(content, &cyclonedxSbom); err != nil {
		t.Fatalf("Failed to parse generated CycloneDX SBOM: %v", err)
	}

	if cyclonedxSbom.SpecVersion == "" {
		t.Error("Generated CycloneDX SBOM missing spec version")
	}

	if len(cyclonedxSbom.Components) == 0 {
		t.Error("Generated CycloneDX SBOM contains no components")
	}
}

func TestCosignSignatureValidation(t *testing.T) {
	// Check if cosign is available
	if _, err := exec.LookPath("cosign"); err != nil {
		t.Skip("cosign not available, skipping signature validation test")
	}

	// Test cosign verify command structure (without actual verification)
	testCases := []struct {
		name     string
		imageRef string
		args     []string
	}{
		{
			name:     "Basic signature verification",
			imageRef: "ghcr.io/agentflow/agentflow/control-plane:latest",
			args: []string{
				"verify",
				"--certificate-identity-regexp=https://github.com/agentflow/agentflow",
				"--certificate-oidc-issuer=https://token.actions.githubusercontent.com",
			},
		},
		{
			name:     "Attestation verification",
			imageRef: "ghcr.io/agentflow/agentflow/worker:latest",
			args: []string{
				"verify-attestation",
				"--certificate-identity-regexp=https://github.com/agentflow/agentflow",
				"--certificate-oidc-issuer=https://token.actions.githubusercontent.com",
				"--type", "slsaprovenance",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Validate command structure
			if len(tc.args) == 0 {
				t.Error("No arguments provided for cosign command")
			}

			// Check that required arguments are present
			argsStr := strings.Join(tc.args, " ")
			if !strings.Contains(argsStr, "--certificate-identity-regexp") {
				t.Error("Missing certificate identity regexp argument")
			}

			if !strings.Contains(argsStr, "--certificate-oidc-issuer") {
				t.Error("Missing certificate OIDC issuer argument")
			}

			// For attestation verification, check type is specified
			if strings.Contains(argsStr, "verify-attestation") && !strings.Contains(argsStr, "--type") {
				t.Error("Missing attestation type for verify-attestation command")
			}
		})
	}
}

func TestValidationScriptExecution(t *testing.T) {
	// Test that validation scripts exist and are executable
	scripts := []string{
		"validate-sbom-provenance.sh",
		"validate-sbom-provenance.ps1",
	}

	for _, script := range scripts {
		t.Run(script, func(t *testing.T) {
			// Try both current directory and scripts subdirectory
			scriptPaths := []string{
				script,                                 // Current directory
				filepath.Join("scripts", script),       // Scripts subdirectory
				filepath.Join("..", "scripts", script), // Parent/scripts directory
			}

			var scriptPath string
			var found bool
			for _, path := range scriptPaths {
				if _, err := os.Stat(path); err == nil {
					scriptPath = path
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Validation script not found in any expected location: %s", script)
				return
			}

			// Check if script is readable
			content, err := os.ReadFile(scriptPath)
			if err != nil {
				t.Errorf("Cannot read validation script: %s", scriptPath)
				return
			}

			// Basic content validation
			contentStr := string(content)
			if len(contentStr) == 0 {
				t.Errorf("Validation script is empty: %s", scriptPath)
			}

			// Check for required functions/commands
			requiredElements := []string{
				"syft",
				"cosign",
				"SBOM",
				"provenance",
			}

			for _, element := range requiredElements {
				if !strings.Contains(contentStr, element) {
					t.Errorf("Validation script %s missing required element: %s", scriptPath, element)
				}
			}
		})
	}
}

func TestExistingSBOMFile(t *testing.T) {
	// Test the existing af-sbom.spdx.json file
	sbomFile := "af-sbom.spdx.json"

	if _, err := os.Stat(sbomFile); os.IsNotExist(err) {
		t.Skip("af-sbom.spdx.json does not exist, skipping validation")
	}

	content, err := os.ReadFile(sbomFile)
	if err != nil {
		t.Fatalf("Failed to read existing SBOM file: %v", err)
	}

	var sbom SPDXDocument
	if err := json.Unmarshal(content, &sbom); err != nil {
		t.Fatalf("Failed to parse existing SBOM file: %v", err)
	}

	// Validate structure
	if sbom.SPDXVersion == "" {
		t.Error("Existing SBOM missing SPDX version")
	}

	if len(sbom.Packages) == 0 {
		t.Error("Existing SBOM contains no packages")
	}

	if sbom.SPDXID == "" {
		t.Error("Existing SBOM missing SPDX ID")
	}

	// Validate that it contains expected AgentFlow components
	foundAgentFlow := false
	for _, pkg := range sbom.Packages {
		if strings.Contains(pkg.Name, "agentflow") || strings.Contains(pkg.Name, "af") {
			foundAgentFlow = true
			break
		}
	}

	if !foundAgentFlow {
		t.Error("Existing SBOM does not contain AgentFlow components")
	}
}

func TestProvenanceAttestationStructure(t *testing.T) {
	// Test provenance attestation structure validation
	// This tests the expected structure without actual attestation verification

	type ProvenanceAttestation struct {
		PayloadType string `json:"payloadType"`
		Payload     string `json:"payload"`
		Signatures  []struct {
			Keyid     string `json:"keyid"`
			Signature string `json:"sig"`
		} `json:"signatures"`
	}

	type SLSAProvenance struct {
		Type    string `json:"_type"`
		Subject []struct {
			Name   string            `json:"name"`
			Digest map[string]string `json:"digest"`
		} `json:"subject"`
		PredicateType string `json:"predicateType"`
		Predicate     struct {
			Builder struct {
				ID string `json:"id"`
			} `json:"builder"`
			BuildType  string `json:"buildType"`
			Invocation struct {
				ConfigSource struct {
					URI        string            `json:"uri"`
					Digest     map[string]string `json:"digest"`
					EntryPoint string            `json:"entryPoint"`
				} `json:"configSource"`
			} `json:"invocation"`
		} `json:"predicate"`
	}

	// Test that we can create valid provenance structure
	provenance := SLSAProvenance{
		Type: "https://in-toto.io/Statement/v0.1",
		Subject: []struct {
			Name   string            `json:"name"`
			Digest map[string]string `json:"digest"`
		}{
			{
				Name: "ghcr.io/agentflow/agentflow/control-plane",
				Digest: map[string]string{
					"sha256": "abc123",
				},
			},
		},
		PredicateType: "https://slsa.dev/provenance/v0.2",
		Predicate: struct {
			Builder struct {
				ID string `json:"id"`
			} `json:"builder"`
			BuildType  string `json:"buildType"`
			Invocation struct {
				ConfigSource struct {
					URI        string            `json:"uri"`
					Digest     map[string]string `json:"digest"`
					EntryPoint string            `json:"entryPoint"`
				} `json:"configSource"`
			} `json:"invocation"`
		}{
			Builder: struct {
				ID string `json:"id"`
			}{
				ID: "https://github.com/actions/runner",
			},
			BuildType: "https://github.com/actions/workflow",
			Invocation: struct {
				ConfigSource struct {
					URI        string            `json:"uri"`
					Digest     map[string]string `json:"digest"`
					EntryPoint string            `json:"entryPoint"`
				} `json:"configSource"`
			}{
				ConfigSource: struct {
					URI        string            `json:"uri"`
					Digest     map[string]string `json:"digest"`
					EntryPoint string            `json:"entryPoint"`
				}{
					URI: "git+https://github.com/agentflow/agentflow",
					Digest: map[string]string{
						"sha1": "def456",
					},
					EntryPoint: ".github/workflows/container-build.yml",
				},
			},
		},
	}

	// Validate structure
	if provenance.Type == "" {
		t.Error("Provenance missing type")
	}

	if len(provenance.Subject) == 0 {
		t.Error("Provenance missing subject")
	}

	if provenance.PredicateType == "" {
		t.Error("Provenance missing predicate type")
	}

	if provenance.Predicate.Builder.ID == "" {
		t.Error("Provenance missing builder ID")
	}

	// Test JSON marshaling
	_, err := json.Marshal(provenance)
	if err != nil {
		t.Errorf("Failed to marshal provenance structure: %v", err)
	}
}
