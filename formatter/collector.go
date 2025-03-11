package formatter

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/pkg/errors"

	"goimporter/entities"
)

// CollectImports extracts all import statements from Go source code.
func CollectImports(code []byte) ([]entities.Import, error) {
	var allImports []entities.Import

	scanner := bufio.NewScanner(bytes.NewReader(code))
	inImportBlock := false
	foundFirstBlock := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Detect import block boundaries.
		if trimmedLine == "import (" {
			if !foundFirstBlock {
				foundFirstBlock = true
				inImportBlock = true
				continue
			}
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
				allImports = append(allImports, entities.Import{Alias: alias, Path: importPath})
			}
		}
	}

	err := scanner.Err()
	if err != nil {
		return nil, errors.Wrap(err, "scanning imports")
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
