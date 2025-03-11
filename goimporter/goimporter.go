// Package goimporter implements a Go import grouping and sorting tool.
//
// It organizes imports into logical groups based on their origin:
// 1. Standard library.
// 2. External dependencies.
// 3. Internal monorepo packages.
//
// Usage:
//
//	goimporter [flags] [files...]
//
// Flags:
//
//	-r            Process recursively.
//	-d            Dry run mode.
//	-pkgs strings Custom prefix paths to organize (comma-separated).
package goimporter

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	// Standard package prefixes.
	monorepoPrefix         = "gitlab.mvk.com/go/vkgo"
	monorepoCommonPrefix   = "gitlab.mvk.com/go/vkgo/pkg"
	monorepoDomainPrefix   = "gitlab.mvk.com/go/vkgo/projects/health/pkg"
	healthProjectsTemplate = "gitlab.mvk.com/go/vkgo/projects/health/%s"

	// Additional repository prefixes.
	vkApiSdkPrefix = "gitlab.mvk.com/vkapi/vk-go-sdk-private"

	// Common organization prefix for any monorepo.
	orgPrefix = "gitlab.mvk.com"
)

// Config holds the configuration for the import processor.
type Config struct {
	Dir         string
	Recursive   bool
	DryRun      bool
	ExcludeMock bool
	PkgPrefixes []string
}

// Import represents a single import statement.
type Import struct {
	Alias string
	Path  string
}

// ImportGroups organizes imports into logical groups.
type ImportGroups struct {
	Stdlib          []Import // Standard library packages.
	External        []Import // External dependencies.
	MonoRepoCommon  []Import // Common monorepo packages (vkgo/pkg/*).
	MonoRepoDomain  []Import // Domain packages (health/pkg/*).
	MonoRepoOther   []Import // Other monorepo packages.
	ProjectPkg      []Import // Project-specific pkg packages.
	ProjectInternal []Import // Project-specific internal packages.
}

// ParseFlags parses command line arguments into a Config.
func ParseFlags() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.Dir, "dir", ".", "Directory to process")
	flag.BoolVar(&cfg.Recursive, "r", false, "Process files recursively")
	flag.BoolVar(&cfg.DryRun, "d", false, "Don't write changes, just report")
	flag.BoolVar(&cfg.ExcludeMock, "exclude-mock", true, "Exclude mock files")

	customPkgs := flag.String("pkgs", "", "Custom package prefixes (comma-separated)")

	flag.Parse()

	if *customPkgs != "" {
		cfg.PkgPrefixes = strings.Split(*customPkgs, ",")
	}

	return cfg
}

// ExtractProjectName extracts the health project name from a file path.
func ExtractProjectName(filePath string) string {
	// Match health project name from path like gitlab.mvk.com/go/vkgo/projects/health/{project_name}.
	re := regexp.MustCompile(`gitlab\.mvk\.com/go/vkgo/projects/health/([^/]+)`)
	matches := re.FindStringSubmatch(filePath)

	if len(matches) > 1 {
		if matches[1] == "pkg" {
			return ""
		}
		return matches[1]
	}
	return ""
}

// GetImportPrefixes returns the ordered list of import prefixes to use for grouping.
func GetImportPrefixes(filePath string) []string {
	projectName := ExtractProjectName(filePath)
	prefixes := []string{
		monorepoPrefix,
		monorepoCommonPrefix,
		monorepoDomainPrefix,
	}

	if projectName != "" {
		// Project-specific prefixes (these are used for grouping only).
		projectPkgPrefix := fmt.Sprintf(healthProjectsTemplate, projectName) + "/pkg"
		projectInternalPrefix := fmt.Sprintf(healthProjectsTemplate, projectName) + "/internal"

		prefixes = append(prefixes, projectPkgPrefix, projectInternalPrefix)
	}

	return prefixes
}

// ProcessFile organizes imports in a single Go file.
func ProcessFile(filename string, cfg *Config) error {
	code, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	// Check if this is a generated file - if so, skip it.
	if isGeneratedFile(code) {
		fmt.Printf("Skipping generated file: %s\n", filename)
		return nil
	}

	// Get prefixes for this file.
	prefixes := GetImportPrefixes(filename)
	if len(cfg.PkgPrefixes) > 0 {
		prefixes = cfg.PkgPrefixes
	}

	// Collect all imports from the file.
	allImports, err := CollectImports(code)
	if err != nil {
		return fmt.Errorf("collecting imports: %w", err)
	}

	// If no imports were found, nothing to do.
	if len(allImports) == 0 {
		return nil
	}

	// Group imports and remove duplicates.
	groups := GroupImports(allImports, prefixes)

	// Generate the new file content.
	newContent, err := RewriteFile(code, groups)
	if err != nil {
		return fmt.Errorf("rewriting file: %w", err)
	}

	// Skip writing if content didn't change.
	if bytes.Equal(code, newContent) {
		return nil
	}

	// Only write changes if not in dry run mode.
	if !cfg.DryRun {
		if err := os.WriteFile(filename, newContent, 0o644); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}
		fmt.Printf("Processed: %s\n", filename)
	} else {
		fmt.Printf("Would process: %s\n", filename)
	}

	return nil
}

// isGeneratedFile checks if a file is generated based on its first few lines.
func isGeneratedFile(code []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(code))

	// Check first few lines (more reliable than just the first line).
	lineCount := 0
	for scanner.Scan() && lineCount < 5 {
		line := scanner.Text()
		lineCount++

		// Skip empty lines.
		if strings.TrimSpace(line) == "" {
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

		// If we've found the package declaration without finding any generated markers,
		// it's most likely not a generated file.
		if strings.HasPrefix(strings.TrimSpace(line), "package ") {
			return false
		}
	}

	return false
}

// CollectImports extracts all import statements from Go source code.
func CollectImports(code []byte) ([]Import, error) {
	var allImports []Import

	scanner := bufio.NewScanner(bytes.NewReader(code))
	inImportBlock := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Detect import block boundaries.
		if trimmedLine == "import (" {
			inImportBlock = true
			continue
		}

		if inImportBlock && trimmedLine == ")" {
			inImportBlock = false
			continue
		}

		// Process import lines.
		if inImportBlock && trimmedLine != "" && !strings.HasPrefix(trimmedLine, "//") {
			// Extract import path and alias.
			importPath, alias := parseImportLine(trimmedLine)

			if importPath != "" {
				allImports = append(allImports, Import{Alias: alias, Path: importPath})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning imports: %w", err)
	}

	return allImports, nil
}

// parseImportLine extracts import path and optional alias from a line.
func parseImportLine(line string) (path, alias string) {
	// Extract the path from the quotes.
	startQuote := strings.Index(line, "\"")
	endQuote := strings.LastIndex(line, "\"")

	if startQuote != -1 && endQuote != -1 && endQuote > startQuote {
		path = line[startQuote+1 : endQuote]

		// Check for alias before the quoted path.
		beforeQuote := strings.TrimSpace(line[:startQuote])
		if beforeQuote != "" {
			alias = beforeQuote
		}

		return path, alias
	}

	return "", ""
}

// GroupImports organizes imports into logical groups and removes duplicates.
func GroupImports(imports []Import, prefixes []string) ImportGroups {
	groups := ImportGroups{}

	// Track processed paths to avoid duplicates.
	processed := make(map[string]struct{})

	// Collect all project paths for better detection
	projectImports := make(map[string]bool)
	for _, imp := range imports {
		if strings.Contains(imp.Path, "/projects/health/") {
			projectImports[imp.Path] = true
		}
	}

	// Current project detection based on the first project-specific import.
	projectPrefix := ""
	projectName := ""
	for _, imp := range imports {
		if strings.Contains(imp.Path, "/projects/health/") &&
			(strings.Contains(imp.Path, "/internal/") || strings.Contains(imp.Path, "/pkg/")) {
			parts := strings.Split(imp.Path, "/")
			for i, part := range parts {
				if part == "health" && i+1 < len(parts) {
					projectName = parts[i+1]
					if projectName != "pkg" {
						projectPrefix = strings.Join(parts[:i+2], "/")
						break
					}
				}
			}
			if projectPrefix != "" {
				break
			}
		}
	}

	// Project pkg and internal prefixes if a project was detected.
	projectPkgPrefix := ""
	projectInternalPrefix := ""
	if projectPrefix != "" {
		projectPkgPrefix = projectPrefix + "/pkg"
		projectInternalPrefix = projectPrefix + "/internal"
	}

	// Better detection for project packages with specific patterns
	isProjectPkg := func(path string) bool {
		return projectPkgPrefix != "" &&
			strings.HasPrefix(path, projectPrefix) &&
			strings.Contains(path, "/pkg/")
	}

	isProjectInternal := func(path string) bool {
		return projectInternalPrefix != "" &&
			strings.HasPrefix(path, projectPrefix) &&
			strings.Contains(path, "/internal/")
	}

	for _, imp := range imports {
		// Skip duplicates.
		if _, exists := processed[imp.Path]; exists {
			continue
		}
		processed[imp.Path] = struct{}{}

		// Strict import classification.
		switch {
		case !strings.Contains(imp.Path, "."):
			// Standard library (no dots in path).
			groups.Stdlib = append(groups.Stdlib, imp)

		case !strings.HasPrefix(imp.Path, orgPrefix):
			// External packages (not from our organization).
			groups.External = append(groups.External, imp)

		case strings.HasPrefix(imp.Path, monorepoCommonPrefix) ||
			strings.HasPrefix(imp.Path, vkApiSdkPrefix) ||
			(strings.HasPrefix(imp.Path, orgPrefix) &&
				!strings.HasPrefix(imp.Path, monorepoPrefix)):
			// Common monorepo packages:
			// 1. vkgo/pkg/* packages,
			// 2. vkapi/vk-go-sdk-private/* packages,
			// 3. Any other gitlab.mvk.com/* packages (except known monorepo paths).
			groups.MonoRepoCommon = append(groups.MonoRepoCommon, imp)

		case strings.HasPrefix(imp.Path, monorepoDomainPrefix):
			// Domain packages (health/pkg/*).
			groups.MonoRepoDomain = append(groups.MonoRepoDomain, imp)

		case isProjectPkg(imp.Path):
			// Project-specific pkg packages (steps/pkg/*).
			groups.ProjectPkg = append(groups.ProjectPkg, imp)

		case isProjectInternal(imp.Path):
			// Project-specific internal packages.
			groups.ProjectInternal = append(groups.ProjectInternal, imp)

		default:
			// Other monorepo packages.
			groups.MonoRepoOther = append(groups.MonoRepoOther, imp)
		}
	}

	// Sort all groups.
	sortImports(groups.Stdlib)
	sortImports(groups.External)
	sortImports(groups.MonoRepoCommon)
	sortImports(groups.MonoRepoDomain)
	sortImports(groups.MonoRepoOther)
	sortImports(groups.ProjectPkg)
	sortImports(groups.ProjectInternal)

	return groups
}

// sortImports sorts imports alphabetically by path.
func sortImports(imports []Import) {
	sort.Slice(imports, func(i, j int) bool {
		return imports[i].Path < imports[j].Path
	})
}

// RewriteFile generates a new file with organized imports.
func RewriteFile(code []byte, groups ImportGroups) ([]byte, error) {
	var buf bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(code))

	foundFirstImport := false
	inImportBlock := false
	skipImportLines := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "import (" {
			// Only process the first import block we find.
			if !foundFirstImport {
				foundFirstImport = true
				inImportBlock = true

				// Write the import statement with grouped imports.
				indent := strings.TrimSuffix(line, "import (")
				buf.WriteString(line + "\n")

				// Helper function to write a group of imports with proper formatting.
				writeImportGroup := func(imports []Import, needNewline bool) {
					if len(imports) == 0 {
						return
					}

					if needNewline {
						buf.WriteString("\n")
					}

					for _, imp := range imports {
						if imp.Alias != "" {
							buf.WriteString(fmt.Sprintf("%s\t%s %q\n", indent, imp.Alias, imp.Path))
						} else {
							buf.WriteString(fmt.Sprintf("%s\t%q\n", indent, imp.Path))
						}
					}
				}

				// Write all groups with proper separation.
				hasContent := false

				// 1. Standard library.
				writeImportGroup(groups.Stdlib, false)
				hasContent = len(groups.Stdlib) > 0

				// 2. External packages.
				writeImportGroup(groups.External, hasContent)
				hasContent = hasContent || len(groups.External) > 0

				// 3. Common monorepo packages.
				writeImportGroup(groups.MonoRepoCommon, hasContent)
				hasContent = hasContent || len(groups.MonoRepoCommon) > 0

				// 4. Domain packages.
				writeImportGroup(groups.MonoRepoDomain, hasContent)
				hasContent = hasContent || len(groups.MonoRepoDomain) > 0

				// 5. Other monorepo packages.
				writeImportGroup(groups.MonoRepoOther, hasContent)
				hasContent = hasContent || len(groups.MonoRepoOther) > 0

				// 6. Project-specific pkg packages - important: these come before internal.
				writeImportGroup(groups.ProjectPkg, hasContent)
				hasContent = hasContent || len(groups.ProjectPkg) > 0

				// 7. Project-specific internal packages.
				writeImportGroup(groups.ProjectInternal, hasContent)
			} else {
				// Skip additional import blocks.
				skipImportLines = true
			}
			continue
		}

		// End of an import block.
		if inImportBlock && trimmedLine == ")" {
			inImportBlock = false
			buf.WriteString(line + "\n")
			continue
		}

		// Skip lines inside additional import blocks.
		if skipImportLines {
			if trimmedLine == ")" {
				skipImportLines = false
			}
			continue
		}

		// Skip lines inside the first import block (already processed).
		if inImportBlock {
			continue
		}

		// Write all other lines.
		buf.WriteString(line + "\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning file: %w", err)
	}

	return buf.Bytes(), nil
}

// ProcessGoFiles processes all Go files in a directory or recursively.
func ProcessGoFiles(cfg *Config) error {
	if cfg.Recursive {
		return filepath.WalkDir(cfg.Dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.IsDir() && strings.HasSuffix(path, ".go") {
				if cfg.ExcludeMock && strings.Contains(path, "mock") {
					return nil
				}

				if err := ProcessFile(path, cfg); err != nil {
					fmt.Printf("Error processing %s: %v\n", path, err)
				}
			}
			return nil
		})
	}

	// Process only go files in the specified directory.
	entries, err := os.ReadDir(cfg.Dir)
	if err != nil {
		return fmt.Errorf("reading directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			if cfg.ExcludeMock && strings.Contains(entry.Name(), "mock") {
				continue
			}

			path := filepath.Join(cfg.Dir, entry.Name())
			if err := ProcessFile(path, cfg); err != nil {
				fmt.Printf("Error processing %s: %v\n", path, err)
			}
		}
	}

	return nil
}
