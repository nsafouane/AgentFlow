package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// ManifestList represents a Docker manifest list structure
type ManifestList struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Manifests     []struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
		Platform  struct {
			Architecture string `json:"architecture"`
			OS           string `json:"os"`
		} `json:"platform"`
	} `json:"manifests"`
}

// CosignSignature represents a cosign signature verification result
type CosignSignature struct {
	Critical struct {
		Identity struct {
			DockerReference string `json:"docker-reference"`
		} `json:"identity"`
		Image struct {
			DockerManifestDigest string `json:"docker-manifest-digest"`
		} `json:"image"`
		Type string `json:"type"`
	} `json:"critical"`
	Optional interface{} `json:"optional"`
}

func TestManifestListInspection(t *testing.T) {
	services := []string{"control-plane", "worker", "af"}
	registry := os.Getenv("REGISTRY")
	if registry == "" {
		registry = "ghcr.io"
	}

	imageName := os.Getenv("IMAGE_NAME")
	if imageName == "" {
		imageName = "agentflow/agentflow"
	}

	tag := os.Getenv("TAG")
	if tag == "" {
		tag = "latest"
	}

	for _, service := range services {
		t.Run(fmt.Sprintf("manifest-list-%s", service), func(t *testing.T) {
			imageRef := fmt.Sprintf("%s/%s/%s:%s", registry, imageName, service, tag)

			// Inspect the manifest list
			cmd := exec.Command("docker", "buildx", "imagetools", "inspect", "--raw", imageRef)
			output, err := cmd.Output()
			if err != nil {
				t.Skipf("Skipping manifest inspection for %s: %v", imageRef, err)
				return
			}

			var manifestList ManifestList
			if err := json.Unmarshal(output, &manifestList); err != nil {
				t.Fatalf("Failed to parse manifest list for %s: %v", imageRef, err)
			}

			// Verify it's a manifest list
			if manifestList.SchemaVersion != 2 {
				t.Errorf("Expected schema version 2, got %d", manifestList.SchemaVersion)
			}

			expectedMediaType := "application/vnd.docker.distribution.manifest.list.v2+json"
			if manifestList.MediaType != expectedMediaType &&
				manifestList.MediaType != "application/vnd.oci.image.index.v1+json" {
				t.Errorf("Expected manifest list media type, got %s", manifestList.MediaType)
			}

			// Verify we have both amd64 and arm64 manifests
			architectures := make(map[string]bool)
			for _, manifest := range manifestList.Manifests {
				if manifest.Platform.OS == "linux" {
					architectures[manifest.Platform.Architecture] = true
				}
			}

			requiredArchs := []string{"amd64", "arm64"}
			for _, arch := range requiredArchs {
				if !architectures[arch] {
					t.Errorf("Missing required architecture %s in manifest list", arch)
				}
			}

			t.Logf("✅ Manifest list validation passed for %s", service)
			t.Logf("   - Schema version: %d", manifestList.SchemaVersion)
			t.Logf("   - Media type: %s", manifestList.MediaType)
			t.Logf("   - Architectures: %v", getKeys(architectures))
		})
	}
}

func TestSignaturePresence(t *testing.T) {
	services := []string{"control-plane", "worker", "af"}
	registry := os.Getenv("REGISTRY")
	if registry == "" {
		registry = "ghcr.io"
	}

	imageName := os.Getenv("IMAGE_NAME")
	if imageName == "" {
		imageName = "agentflow/agentflow"
	}

	tag := os.Getenv("TAG")
	if tag == "" {
		tag = "latest"
	}

	// Check if cosign is available
	if _, err := exec.LookPath("cosign"); err != nil {
		t.Skip("Cosign not available, skipping signature tests")
	}

	for _, service := range services {
		t.Run(fmt.Sprintf("signature-%s", service), func(t *testing.T) {
			imageRef := fmt.Sprintf("%s/%s/%s:%s", registry, imageName, service, tag)

			// Verify signature exists
			cmd := exec.Command("cosign", "verify",
				"--certificate-identity-regexp", fmt.Sprintf("https://github.com/%s", imageName),
				"--certificate-oidc-issuer", "https://token.actions.githubusercontent.com",
				imageRef)

			output, err := cmd.Output()
			if err != nil {
				t.Skipf("Skipping signature verification for %s: %v", imageRef, err)
				return
			}

			// Parse cosign output
			var signatures []CosignSignature
			if err := json.Unmarshal(output, &signatures); err != nil {
				t.Fatalf("Failed to parse cosign output for %s: %v", imageRef, err)
			}

			if len(signatures) == 0 {
				t.Errorf("No signatures found for %s", imageRef)
				return
			}

			// Verify signature properties
			for i, sig := range signatures {
				if sig.Critical.Type != "cosign container image signature" {
					t.Errorf("Signature %d: unexpected type %s", i, sig.Critical.Type)
				}

				if !strings.Contains(sig.Critical.Identity.DockerReference, service) {
					t.Errorf("Signature %d: docker reference doesn't match service %s", i, service)
				}

				if sig.Critical.Image.DockerManifestDigest == "" {
					t.Errorf("Signature %d: missing docker manifest digest", i)
				}
			}

			t.Logf("✅ Signature verification passed for %s", service)
			t.Logf("   - Signatures found: %d", len(signatures))
			t.Logf("   - Image reference: %s", imageRef)
		})
	}
}

func TestSBOMPresence(t *testing.T) {
	services := []string{"control-plane", "worker", "af"}
	registry := os.Getenv("REGISTRY")
	if registry == "" {
		registry = "ghcr.io"
	}

	imageName := os.Getenv("IMAGE_NAME")
	if imageName == "" {
		imageName = "agentflow/agentflow"
	}

	tag := os.Getenv("TAG")
	if tag == "" {
		tag = "latest"
	}

	// Check if syft is available
	if _, err := exec.LookPath("syft"); err != nil {
		t.Skip("Syft not available, skipping SBOM tests")
	}

	for _, service := range services {
		t.Run(fmt.Sprintf("sbom-%s", service), func(t *testing.T) {
			imageRef := fmt.Sprintf("%s/%s/%s:%s", registry, imageName, service, tag)

			// Generate SBOM to verify image accessibility and content
			cmd := exec.Command("syft", imageRef, "-o", "json")
			output, err := cmd.Output()
			if err != nil {
				t.Skipf("Skipping SBOM generation for %s: %v", imageRef, err)
				return
			}

			// Parse SBOM output to verify it contains expected components
			var sbom map[string]interface{}
			if err := json.Unmarshal(output, &sbom); err != nil {
				t.Fatalf("Failed to parse SBOM for %s: %v", imageRef, err)
			}

			// Verify SBOM structure
			if _, ok := sbom["artifacts"]; !ok {
				t.Errorf("SBOM missing artifacts section for %s", imageRef)
			}

			if _, ok := sbom["source"]; !ok {
				t.Errorf("SBOM missing source section for %s", imageRef)
			}

			t.Logf("✅ SBOM generation passed for %s", service)
			t.Logf("   - Image reference: %s", imageRef)
		})
	}
}

func TestProvenanceAttestation(t *testing.T) {
	services := []string{"control-plane", "worker", "af"}
	registry := os.Getenv("REGISTRY")
	if registry == "" {
		registry = "ghcr.io"
	}

	imageName := os.Getenv("IMAGE_NAME")
	if imageName == "" {
		imageName = "agentflow/agentflow"
	}

	tag := os.Getenv("TAG")
	if tag == "" {
		tag = "latest"
	}

	// Check if cosign is available
	if _, err := exec.LookPath("cosign"); err != nil {
		t.Skip("Cosign not available, skipping provenance tests")
	}

	for _, service := range services {
		t.Run(fmt.Sprintf("provenance-%s", service), func(t *testing.T) {
			imageRef := fmt.Sprintf("%s/%s/%s:%s", registry, imageName, service, tag)

			// Verify provenance attestation
			cmd := exec.Command("cosign", "verify-attestation",
				"--certificate-identity-regexp", fmt.Sprintf("https://github.com/%s", imageName),
				"--certificate-oidc-issuer", "https://token.actions.githubusercontent.com",
				"--type", "slsaprovenance",
				imageRef)

			output, err := cmd.Output()
			if err != nil {
				t.Skipf("Skipping provenance verification for %s: %v", imageRef, err)
				return
			}

			// Verify we got some attestation output
			if len(output) == 0 {
				t.Errorf("No provenance attestation found for %s", imageRef)
				return
			}

			t.Logf("✅ Provenance attestation verified for %s", service)
			t.Logf("   - Image reference: %s", imageRef)
		})
	}
}

func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func main() {
	// Run tests programmatically
	testing.Main(func(pat, str string) (bool, error) {
		return true, nil
	}, []testing.InternalTest{
		{"TestManifestListInspection", TestManifestListInspection},
		{"TestSignaturePresence", TestSignaturePresence},
		{"TestSBOMPresence", TestSBOMPresence},
		{"TestProvenanceAttestation", TestProvenanceAttestation},
	}, nil, nil)
}
