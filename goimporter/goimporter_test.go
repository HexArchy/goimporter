package goimporter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportOrdering(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "standard library imports only",
			input: `package test

import (
    "fmt"
    "strings"
    "context"
)

func main() {
    fmt.Println("test")
}
`,
			expected: `package test

import (
    "context"
    "fmt"
    "strings"
)

func main() {
    fmt.Println("test")
}
`,
		},
		{
			name: "mixed standard and external imports",
			input: `package test

import (
    "strings"
    "github.com/pkg/errors"
    "context"
    "time"
)

func main() {
    fmt.Println("test")
}
`,
			expected: `package test

import (
    "context"
    "strings"
    "time"

    "github.com/pkg/errors"
)

func main() {
    fmt.Println("test")
}
`,
		},
		{
			name: "project pkg before internal imports",
			input: `package test

import (
    "context"

    "github.com/pkg/errors"

    "gitlab.mvk.com/go/vkgo/projects/health/pkg/richerr"

    seasonuserentities "gitlab.mvk.com/go/vkgo/projects/health/steps/internal/core/aggregates/season-user/entities"
    "gitlab.mvk.com/go/vkgo/projects/health/steps/internal/core/sharedentities"
    "gitlab.mvk.com/go/vkgo/projects/health/steps/pkg/ntime"
)

func main() {
    // Test function
}
`,
			expected: `package test

import (
    "context"

    "github.com/pkg/errors"

    "gitlab.mvk.com/go/vkgo/projects/health/pkg/richerr"

    "gitlab.mvk.com/go/vkgo/projects/health/steps/pkg/ntime"

    seasonuserentities "gitlab.mvk.com/go/vkgo/projects/health/steps/internal/core/aggregates/season-user/entities"
    "gitlab.mvk.com/go/vkgo/projects/health/steps/internal/core/sharedentities"
)

func main() {
    // Test function
}
`,
		},
		{
			name: "complex import mix",
			input: `package test

import (
    "context"
    "fmt"
    "strings"
    "github.com/pkg/errors"
    "testing"
    "time"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "pgregory.net/rand"
    "gitlab.mvk.com/go/vkgo/pkg/paas/memcache"
    "gitlab.mvk.com/go/vkgo/pkg/rpc"
    "gitlab.mvk.com/go/vkgo/pkg/vktl/gen/tlnews2"
    "github.com/golang/mock/gomock"
    "gitlab.mvk.com/go/vkgo/pkg/vktl/gen/tlwall"
    "strconv"
    "gitlab.mvk.com/go/vkgo/projects/health/pkg/richerr"
    "gitlab.mvk.com/go/vkgo/projects/health/feed/internal/context/aggregates/feed-object/domain"
)

func TestSomething(t *testing.T) {
    // Test function
}
`,
			expected: `package test

import (
    "context"
    "fmt"
    "strconv"
    "strings"
    "testing"
    "time"

    "github.com/golang/mock/gomock"
    "github.com/google/uuid"
    "github.com/pkg/errors"
    "github.com/stretchr/testify/assert"
    "pgregory.net/rand"

    "gitlab.mvk.com/go/vkgo/pkg/paas/memcache"
    "gitlab.mvk.com/go/vkgo/pkg/rpc"
    "gitlab.mvk.com/go/vkgo/pkg/vktl/gen/tlnews2"
    "gitlab.mvk.com/go/vkgo/pkg/vktl/gen/tlwall"

    "gitlab.mvk.com/go/vkgo/projects/health/pkg/richerr"

    "gitlab.mvk.com/go/vkgo/projects/health/feed/internal/context/aggregates/feed-object/domain"
)

func TestSomething(t *testing.T) {
    // Test function
}
`,
		},
		{
			name: "mixed package types with multiple project imports",
			input: `package test

import (
    "context"
    "github.com/google/uuid"
    "github.com/pkg/errors"
    "gitlab.mvk.com/go/vkgo/pkg/meowdb/meowql"
    "gitlab.mvk.com/go/vkgo/pkg/paas/meowdb"
    "gitlab.mvk.com/go/vkgo/projects/health/pkg/meowdb/forward"
    "gitlab.mvk.com/go/vkgo/projects/health/workouts/internal/core/entities"
    "gitlab.mvk.com/go/vkgo/projects/health/courses/pkg/convert"
)

func main() {
    // Test function with multiple project imports
}
`,
			expected: `package test

import (
    "context"

    "github.com/google/uuid"
    "github.com/pkg/errors"

    "gitlab.mvk.com/go/vkgo/pkg/meowdb/meowql"
    "gitlab.mvk.com/go/vkgo/pkg/paas/meowdb"

    "gitlab.mvk.com/go/vkgo/projects/health/pkg/meowdb/forward"

    "gitlab.mvk.com/go/vkgo/projects/health/courses/pkg/convert"

    "gitlab.mvk.com/go/vkgo/projects/health/workouts/internal/core/entities"
)

func main() {
    // Test function with multiple project imports
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file with the test input
			tempDir := t.TempDir()
			tempFile := filepath.Join(tempDir, "test.go")

			err := os.WriteFile(tempFile, []byte(tt.input), 0o644)
			if err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}

			// Process the file with our tool
			cfg := &Config{
				Dir:         tempDir,
				Recursive:   false,
				DryRun:      false,
				ExcludeMock: true,
			}

			err = ProcessFile(tempFile, cfg)
			if err != nil {
				t.Fatalf("ProcessFile() error = %v", err)
			}

			// Read the modified file
			output, err := os.ReadFile(tempFile)
			if err != nil {
				t.Fatalf("Failed to read temp file: %v", err)
			}

			// Compare with expected output
			if normalizeWhitespace(string(output)) != normalizeWhitespace(tt.expected) {
				t.Errorf("Import grouping failed.\nExpected:\n%s\nGot:\n%s", tt.expected, string(output))
			}
		})
	}
}

// TestImportDetection tests the ability to correctly detect and extract import paths.
func TestImportDetection(t *testing.T) {
	imports := []struct {
		line      string
		wantPath  string
		wantAlias string
	}{
		{`"fmt"`, "fmt", ""},
		{`alias "fmt"`, "fmt", "alias"},
		{`somealias "gitlab.mvk.com/go/vkgo/projects/health/steps/pkg/ntime"`, "gitlab.mvk.com/go/vkgo/projects/health/steps/pkg/ntime", "somealias"},
		{` "gitlab.mvk.com/go/vkgo/projects/health/steps/pkg/ntime"`, "gitlab.mvk.com/go/vkgo/projects/health/steps/pkg/ntime", ""},
	}

	for _, tt := range imports {
		t.Run(tt.line, func(t *testing.T) {
			path, alias := parseImportLine(tt.line)
			if path != tt.wantPath {
				t.Errorf("parseImportLine() path = %v, want %v", path, tt.wantPath)
			}
			if alias != tt.wantAlias {
				t.Errorf("parseImportLine() alias = %v, want %v", alias, tt.wantAlias)
			}
		})
	}
}

// TestProjectDetection tests the ability to detect project names from file paths.
func TestProjectDetection(t *testing.T) {
	paths := []struct {
		path string
		want string
	}{
		{"gitlab.mvk.com/go/vkgo/projects/health/steps/pkg/ntime/time.go", "steps"},
		{"gitlab.mvk.com/go/vkgo/projects/health/pkg/richerr/errors.go", ""},
		{"gitlab.mvk.com/go/vkgo/projects/health/feed/internal/repo/repository.go", "feed"},
		{"/path/to/gitlab.mvk.com/go/vkgo/projects/health/steps/internal/handler.go", "steps"},
		{"/nonsense/path/file.go", ""},
	}

	for _, tt := range paths {
		t.Run(tt.path, func(t *testing.T) {
			got := ExtractProjectName(tt.path)
			if got != tt.want {
				t.Errorf("ExtractProjectName() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGroupImports tests that imports are grouped correctly.
func TestGroupImports(t *testing.T) {
	testImports := []Import{
		{Path: "context"},
		{Path: "strings"},
		{Path: "github.com/pkg/errors"},
		{Path: "gitlab.mvk.com/go/vkgo/pkg/rpc"},
		{Path: "gitlab.mvk.com/go/vkgo/projects/health/pkg/richerr"},
		{Path: "gitlab.mvk.com/go/vkgo/projects/health/steps/pkg/ntime"},
		{Alias: "seasonuserentities", Path: "gitlab.mvk.com/go/vkgo/projects/health/steps/internal/core/aggregates/season-user/entities"},
	}

	// Create test file path that should be recognized as part of the steps project
	testFilePath := "/path/to/gitlab.mvk.com/go/vkgo/projects/health/steps/cmd/main.go"
	prefixes := GetImportPrefixes(testFilePath)

	groups := GroupImports(testImports, prefixes)

	// Check that stdlib is correct
	if len(groups.Stdlib) != 2 {
		t.Errorf("Expected 2 stdlib imports, got %d", len(groups.Stdlib))
	}

	// Check that external is correct
	if len(groups.External) != 1 || groups.External[0].Path != "github.com/pkg/errors" {
		t.Errorf("External imports not grouped correctly: %v", groups.External)
	}

	// Check that project pkg comes before project internal
	if len(groups.ProjectPkg) != 1 || !strings.Contains(groups.ProjectPkg[0].Path, "/pkg/ntime") {
		t.Errorf("Project pkg imports not grouped correctly: %v", groups.ProjectPkg)
	}

	if len(groups.ProjectInternal) != 1 || !strings.Contains(groups.ProjectInternal[0].Path, "/internal/") {
		t.Errorf("Project internal imports not grouped correctly: %v", groups.ProjectInternal)
	}
}

// TestRewriteFile tests that the file content is rewritten correctly with proper import groups.
func TestRewriteFile(t *testing.T) {
	input := `package test

import (
    "context"
    "strings"
    "gitlab.mvk.com/go/vkgo/projects/health/steps/internal/core/sharedentities"
    "github.com/pkg/errors"
    "gitlab.mvk.com/go/vkgo/projects/health/steps/pkg/ntime"
)

func main() {
    // Test content
}
`
	groups := ImportGroups{
		Stdlib: []Import{
			{Path: "context"},
			{Path: "strings"},
		},
		External: []Import{
			{Path: "github.com/pkg/errors"},
		},
		ProjectPkg: []Import{
			{Path: "gitlab.mvk.com/go/vkgo/projects/health/steps/pkg/ntime"},
		},
		ProjectInternal: []Import{
			{Path: "gitlab.mvk.com/go/vkgo/projects/health/steps/internal/core/sharedentities"},
		},
	}

	expected := `package test

import (
    "context"
    "strings"

    "github.com/pkg/errors"

    "gitlab.mvk.com/go/vkgo/projects/health/steps/pkg/ntime"

    "gitlab.mvk.com/go/vkgo/projects/health/steps/internal/core/sharedentities"
)

func main() {
    // Test content
}
`

	result, err := RewriteFile([]byte(input), groups)
	if err != nil {
		t.Fatalf("RewriteFile() error = %v", err)
	}

	if normalizeWhitespace(string(result)) != normalizeWhitespace(expected) {
		t.Errorf("RewriteFile() returned incorrect content.\nExpected:\n%s\nGot:\n%s", expected, string(result))
	}
}

// Helper function to normalize whitespace for comparing strings
func normalizeWhitespace(s string) string {
	// Replace all whitespace sequences with a single space
	s = strings.Join(strings.Fields(s), " ")
	// Remove spaces after opening and before closing braces/parentheses
	s = strings.ReplaceAll(s, "{ ", "{")
	s = strings.ReplaceAll(s, " }", "}")
	s = strings.ReplaceAll(s, "( ", "(")
	s = strings.ReplaceAll(s, " )", ")")
	return s
}
