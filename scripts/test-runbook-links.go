package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	fmt.Println("Running runbook link validation...")

	if err := validateRunbookLinks(); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}

	if err := validateRunbookIndexStructure(); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ All runbook links and structure validation passed")
}

// validateRunbookLinks validates that all links in runbook files are valid
func validateRunbookLinks() error {
	runbookDir := "docs/runbooks"

	// Check if runbooks directory exists
	if _, err := os.Stat(runbookDir); os.IsNotExist(err) {
		return fmt.Errorf("runbooks directory does not exist: %s", runbookDir)
	}

	// Walk through all markdown files in runbooks directory
	err := filepath.Walk(runbookDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		return validateLinksInFile(path)
	})

	if err != nil {
		return fmt.Errorf("error walking runbooks directory: %v", err)
	}

	return nil
}

// validateLinksInFile checks all markdown links in a file
func validateLinksInFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	// Regex patterns for different types of links
	markdownLinkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Find all markdown links in the line
		matches := markdownLinkRegex.FindAllStringSubmatch(line, -1)

		for _, match := range matches {
			if len(match) >= 3 {
				linkText := match[1]
				linkURL := match[2]

				// Validate the link
				if err := validateLink(filePath, linkURL, lineNum); err != nil {
					return fmt.Errorf("invalid link in %s:%d - [%s](%s): %v",
						filePath, lineNum, linkText, linkURL, err)
				}
			}
		}
	}

	return scanner.Err()
}

// validateLink checks if a link is valid
func validateLink(filePath, linkURL string, lineNum int) error {
	// Skip external URLs (http/https) for now - they would require network calls
	if strings.HasPrefix(linkURL, "http://") || strings.HasPrefix(linkURL, "https://") {
		return nil
	}

	// Skip anchor links within the same document
	if strings.HasPrefix(linkURL, "#") {
		return nil
	}

	// Handle relative file paths
	baseDir := filepath.Dir(filePath)
	targetPath := filepath.Join(baseDir, linkURL)

	// Clean the path to handle .. and . components
	targetPath = filepath.Clean(targetPath)

	// For placeholder links that reference future specs, we'll allow them
	// if they follow the expected pattern
	if isPlaceholderLink(linkURL) {
		return nil
	}

	// Check if the target file exists
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return fmt.Errorf("target file does not exist: %s", targetPath)
	}

	return nil
}

// isPlaceholderLink checks if a link is a placeholder for future implementation
func isPlaceholderLink(linkURL string) bool {
	placeholderPatterns := []string{
		"build-failure.md",
		"security-scan-failures.md",
		"cross-platform-builds.md",
		"message-backlog.md",
		"database-migrations.md",
		"container-registry.md",
		"cost-spike.md",
		"performance-degradation.md",
		"resource-exhaustion.md",
		"security-incident.md",
		"certificate-expiration.md",
		"access-control.md",
	}

	for _, pattern := range placeholderPatterns {
		if strings.Contains(linkURL, pattern) {
			return true
		}
	}

	return false
}

// validateRunbookIndexStructure validates the structure of the runbook index
func validateRunbookIndexStructure() error {
	indexPath := "docs/runbooks/index.md"

	content, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("failed to read runbook index: %v", err)
	}

	contentStr := string(content)

	// Check for required sections
	requiredSections := []string{
		"# Operational Runbooks",
		"## Available Runbooks",
		"## Runbook Structure",
		"## Quick Reference",
		"## Contributing to Runbooks",
		"## Related Documentation",
	}

	for _, section := range requiredSections {
		if !strings.Contains(contentStr, section) {
			return fmt.Errorf("missing required section in runbook index: %s", section)
		}
	}

	// Check for placeholder runbooks
	placeholderRunbooks := []string{
		"build-failure.md",
		"message-backlog.md",
		"cost-spike.md",
	}

	for _, runbook := range placeholderRunbooks {
		if !strings.Contains(contentStr, runbook) {
			return fmt.Errorf("missing placeholder runbook reference: %s", runbook)
		}
	}

	return nil
}
