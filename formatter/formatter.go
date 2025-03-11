package formatter

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"goimporter/entities"
)

// GroupImports organizes imports into logical groups and removes duplicates.
// TODO: Add support for additional import groups with prefixes.
func GroupImports(imports []entities.Import, _ []string, repo *entities.RepoConfig) entities.ImportGroups {
	groups := entities.ImportGroups{}

	// Track processed paths to avoid duplicates.
	processed := make(map[string]struct{})

	// Extract domain part from projects template for path matching.
	domainPart := extractDomainFromTemplate(repo.ProjectsTemplate)

	// Collect all project paths for better detection.
	projectImports := make(map[string]bool)
	for _, imp := range imports {
		if domainPart != "" && strings.Contains(imp.Path, "/projects/"+domainPart+"/") {
			projectImports[imp.Path] = true
		}
	}

	// Current project detection based on the first project-specific import.
	projectPrefix := ""
	projectName := ""
	for _, imp := range imports {
		if domainPart != "" && strings.Contains(imp.Path, "/projects/"+domainPart+"/") &&
			(strings.Contains(imp.Path, "/internal/") || strings.Contains(imp.Path, "/pkg/")) {
			parts := strings.Split(imp.Path, "/")
			for i, part := range parts {
				if part == domainPart && i+1 < len(parts) {
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

	// Better detection for project packages with specific patterns.
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

		case !strings.HasPrefix(imp.Path, repo.OrgPrefix):
			// External packages (not from our organization).
			groups.External = append(groups.External, imp)

		case strings.HasPrefix(imp.Path, repo.CommonPrefix) ||
			stringHasPrefixAny(imp.Path, repo.AdditionalCommonPrefixes) ||
			(strings.HasPrefix(imp.Path, repo.OrgPrefix) &&
				!strings.HasPrefix(imp.Path, repo.RepoPrefix)):
			// Common organization packages:
			// 1. Common packages (e.g. repo/pkg/*),
			// 2. Additional common prefixes from config,
			// 3. Any other organization packages (except known repo paths).
			groups.OrgCommon = append(groups.OrgCommon, imp)

		case strings.HasPrefix(imp.Path, repo.DomainPrefix):
			// Domain packages.
			groups.DomainCommon = append(groups.DomainCommon, imp)

		case isProjectPkg(imp.Path):
			// Project-specific pkg packages.
			groups.ProjectPkg = append(groups.ProjectPkg, imp)

		case isProjectInternal(imp.Path):
			// Project-specific internal packages.
			groups.ProjectInternal = append(groups.ProjectInternal, imp)

		default:
			// Other repository packages.
			groups.RepoOther = append(groups.RepoOther, imp)
		}
	}

	// Sort all groups.
	sortImports(groups.Stdlib)
	sortImports(groups.External)
	sortImports(groups.OrgCommon)
	sortImports(groups.DomainCommon)
	sortImports(groups.RepoOther)
	sortImports(groups.ProjectPkg)
	sortImports(groups.ProjectInternal)

	return groups
}

// sortImports sorts imports alphabetically by path.
func sortImports(imports []entities.Import) {
	sort.Slice(imports, func(i, j int) bool {
		return imports[i].Path < imports[j].Path
	})
}

// stringHasPrefixAny checks if a string has any of the given prefixes.
func stringHasPrefixAny(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

// RewriteFile generates a new file with organized imports.
func RewriteFile(code []byte, groups entities.ImportGroups) ([]byte, error) {
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
				writeImportGroup := func(imports []entities.Import, needNewline bool) {
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

				// 3. Common organization packages.
				writeImportGroup(groups.OrgCommon, hasContent)
				hasContent = hasContent || len(groups.OrgCommon) > 0

				// 4. Domain packages.
				writeImportGroup(groups.DomainCommon, hasContent)
				hasContent = hasContent || len(groups.DomainCommon) > 0

				// 5. Other repository packages.
				writeImportGroup(groups.RepoOther, hasContent)
				hasContent = hasContent || len(groups.RepoOther) > 0

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

	err := scanner.Err()
	if err != nil {
		return nil, errors.Wrap(err, "scanning file")
	}

	return buf.Bytes(), nil
}
