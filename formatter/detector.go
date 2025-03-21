package formatter

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"goimporter/entities"
)

// IsGeneratedFile checks if a file is generated based on its first few lines.
func IsGeneratedFile(code []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(code))

	// Check first few lines (more reliable than just the first line).
	lineCount := 0
	foundPackage := false

	for scanner.Scan() && lineCount < 10 {
		line := scanner.Text()
		lineCount++

		// Skip empty lines.
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check if this is the package declaration.
		if strings.HasPrefix(strings.TrimSpace(line), "package ") {
			foundPackage = true
			continue
		}

		// Generated markers should be before package declaration.
		if foundPackage {
			continue
		}

		// Check for common generated file markers.
		lowercaseLine := strings.ToLower(line)
		if strings.Contains(lowercaseLine, "generated") &&
			(strings.Contains(line, "//") || strings.Contains(line, "/*")) {
			return true
		}

		// Common explicit markers.
		for _, marker := range []string{
			"do not edit",
			"auto-generated",
			"autogenerated",
			"code generated",
			"by mockgen",
			"by protoc",
			"automatically generated",
		} {
			if strings.Contains(lowercaseLine, marker) {
				return true
			}
		}
	}

	return false
}

// ExtractProjectName extracts the project name from a file path.
func ExtractProjectName(filePath string, repo *entities.RepoConfig) string {
	// Extract domain part from projects template.
	domainPart := extractDomainFromTemplate(repo.ProjectsTemplate)
	if domainPart == "" {
		return ""
	}

	// Create a regex pattern to match project name from the path.
	pattern := strings.ReplaceAll(regexp.QuoteMeta(repo.RepoPrefix), "/", "\\/") +
		"\\/projects\\/" + domainPart + "\\/([^\\/]+)"
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(filePath)

	if len(matches) > 1 {
		if matches[1] == "pkg" {
			return ""
		}
		return matches[1]
	}
	return ""
}

// extractDomainFromTemplate extracts the domain name from a projects template.
func extractDomainFromTemplate(template string) string {
	parts := strings.Split(template, "/")
	for i, part := range parts {
		if i > 0 && part == "projects" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// GetImportPrefixes returns the ordered list of import prefixes to use for grouping.
func GetImportPrefixes(filePath string, repo *entities.RepoConfig) []string {
	projectName := ExtractProjectName(filePath, repo)
	prefixes := []string{
		repo.RepoPrefix,
		repo.CommonPrefix,
		repo.DomainPrefix,
	}

	if projectName != "" {
		// Project-specific prefixes (these are used for grouping only).
		projectPkgPrefix := fmt.Sprintf(repo.ProjectsTemplate, projectName) + "/pkg"
		projectInternalPrefix := fmt.Sprintf(repo.ProjectsTemplate, projectName) + "/internal"

		prefixes = append(prefixes, projectPkgPrefix, projectInternalPrefix)
	}

	return prefixes
}
