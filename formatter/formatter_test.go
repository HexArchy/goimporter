package formatter

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"goimporter/config"
	"goimporter/entities"
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
			tempDir := t.TempDir()
			tempFile := filepath.Join(tempDir, "test.go")

			err := os.WriteFile(tempFile, []byte(tt.input), 0o644)
			if err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}

			cfg := &config.Config{
				Dir:         tempDir,
				Recursive:   false,
				DryRun:      false,
				ExcludeMock: true,
				Repo: &entities.RepoConfig{
					OrgPrefix:        "gitlab.mvk.com",
					RepoPrefix:       "gitlab.mvk.com/go/vkgo",
					CommonPrefix:     "gitlab.mvk.com/go/vkgo/pkg",
					DomainPrefix:     "gitlab.mvk.com/go/vkgo/projects/health/pkg",
					ProjectsTemplate: "gitlab.mvk.com/go/vkgo/projects/health/%s",
				},
			}

			err = ProcessFile(tempFile, cfg)
			if err != nil {
				t.Fatalf("ProcessFile() error = %v", err)
			}

			output, err := os.ReadFile(tempFile)
			if err != nil {
				t.Fatalf("Failed to read temp file: %v", err)
			}

			if normalizeOutput(string(output)) != normalizeOutput(tt.expected) {
				t.Errorf("\nImport grouping failed.\nExpected:\n%s\n\nGot:\n%s", tt.expected, string(output))
			}
		})
	}
}

func normalizeOutput(s string) string {
	lines := strings.Split(s, "\n")
	var normalized []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return strings.Join(normalized, "\n")
}

func TestIsGeneratedFile(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
	}{
		{
			name: "not generated",
			code: `package test

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
`,
			want: false,
		},
		{
			name: "generated with marker",
			code: `// Code generated by protoc-gen-go. DO NOT EDIT.
package test

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
`,
			want: true,
		},
		{
			name: "generated with do not edit",
			code: `// DO NOT EDIT.
package test

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
`,
			want: true,
		},
		{
			name: "generated with auto-generated",
			code: `// auto-generated
package test

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
`,
			want: true,
		},
		{
			name: "generated with mockgen",
			code: `// Source: github.com/myorg/pkg/service (interfaces: Service)
// Package mocks is a generated GoMock package.
// Generated by mockgen 1.6.0
package mocks

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
`,
			want: true,
		},
		{
			name: "generated but marker after package",
			code: `package test

// Code generated by protoc-gen-go. DO NOT EDIT.
import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGeneratedFile([]byte(tt.code)); got != tt.want {
				t.Errorf("IsGeneratedFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractProjectName(t *testing.T) {
	repo := &entities.RepoConfig{
		RepoPrefix:       "gitlab.mvk.com/go/vkgo",
		ProjectsTemplate: "gitlab.mvk.com/go/vkgo/projects/health/%s",
	}

	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{
			name:     "valid project path",
			filePath: "/path/to/gitlab.mvk.com/go/vkgo/projects/health/steps/pkg/ntime/time.go",
			want:     "steps",
		},
		{
			name:     "domain pkg path",
			filePath: "/path/to/gitlab.mvk.com/go/vkgo/projects/health/pkg/richerr/errors.go",
			want:     "",
		},
		{
			name:     "project internal path",
			filePath: "/path/to/gitlab.mvk.com/go/vkgo/projects/health/feed/internal/repo/repository.go",
			want:     "feed",
		},
		{
			name:     "not a project path",
			filePath: "/path/to/gitlab.mvk.com/go/vkgo/pkg/common/util.go",
			want:     "",
		},
		{
			name:     "invalid path",
			filePath: "/path/to/something/else.go",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractProjectName(tt.filePath, repo); got != tt.want {
				t.Errorf("ExtractProjectName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractDomainFromTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "standard template",
			template: "gitlab.mvk.com/go/vkgo/projects/health/%s",
			want:     "health",
		},
		{
			name:     "different domain",
			template: "github.com/myorg/myrepo/projects/domain/%s",
			want:     "domain",
		},
		{
			name:     "no projects segment",
			template: "github.com/myorg/myrepo/modules/%s",
			want:     "",
		},
		{
			name:     "empty template",
			template: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractDomainFromTemplate(tt.template); got != tt.want {
				t.Errorf("extractDomainFromTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetImportPrefixes(t *testing.T) {
	repo := &entities.RepoConfig{
		RepoPrefix:       "gitlab.mvk.com/go/vkgo",
		CommonPrefix:     "gitlab.mvk.com/go/vkgo/pkg",
		DomainPrefix:     "gitlab.mvk.com/go/vkgo/projects/health/pkg",
		ProjectsTemplate: "gitlab.mvk.com/go/vkgo/projects/health/%s",
	}

	tests := []struct {
		name     string
		filePath string
		want     []string
	}{
		{
			name:     "file in steps project",
			filePath: "/path/to/gitlab.mvk.com/go/vkgo/projects/health/steps/cmd/main.go",
			want: []string{
				"gitlab.mvk.com/go/vkgo",
				"gitlab.mvk.com/go/vkgo/pkg",
				"gitlab.mvk.com/go/vkgo/projects/health/pkg",
				"gitlab.mvk.com/go/vkgo/projects/health/steps/pkg",
				"gitlab.mvk.com/go/vkgo/projects/health/steps/internal",
			},
		},
		{
			name:     "file in common repository",
			filePath: "/path/to/gitlab.mvk.com/go/vkgo/pkg/common/util.go",
			want: []string{
				"gitlab.mvk.com/go/vkgo",
				"gitlab.mvk.com/go/vkgo/pkg",
				"gitlab.mvk.com/go/vkgo/projects/health/pkg",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetImportPrefixes(tt.filePath, repo); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetImportPrefixes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCollectImports(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		want    []entities.Import
		wantErr bool
	}{
		{
			name: "simple imports",
			code: `package test

import (
    "fmt"
    "strings"
)

func main() {}
`,
			want: []entities.Import{
				{Path: "fmt"},
				{Path: "strings"},
			},
			wantErr: false,
		},
		{
			name: "imports with aliases",
			code: `package test

import (
    "fmt"
    e "errors"
    . "testing"
    _ "database/sql"
)

func main() {}
`,
			want: []entities.Import{
				{Path: "fmt"},
				{Alias: "e", Path: "errors"},
				{Alias: ".", Path: "testing"},
				{Alias: "_", Path: "database/sql"},
			},
			wantErr: false,
		},
		{
			name: "mixed imports with comments",
			code: `package test

import (
    "fmt"
    // This is a comment
    "strings"
    
    // Another comment
    "time"
)

func main() {}
`,
			want: []entities.Import{
				{Path: "fmt"},
				{Path: "strings"},
				{Path: "time"},
			},
			wantErr: false,
		},
		{
			name: "multiple import blocks",
			code: `package test

import (
    "fmt"
    "strings"
)

import (
    "time"
    "context"
)

func main() {}
`,
			want: []entities.Import{
				{Path: "fmt"},
				{Path: "strings"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CollectImports([]byte(tt.code))
			if (err != nil) != tt.wantErr {
				t.Errorf("CollectImports() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CollectImports() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseImportLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantPath  string
		wantAlias string
	}{
		{
			name:      "simple import",
			line:      `"fmt"`,
			wantPath:  "fmt",
			wantAlias: "",
		},
		{
			name:      "import with alias",
			line:      `aliases "fmt"`,
			wantPath:  "fmt",
			wantAlias: "aliases",
		},
		{
			name:      "import with dot alias",
			line:      `. "testing"`,
			wantPath:  "testing",
			wantAlias: ".",
		},
		{
			name:      "import with underscore alias",
			line:      `_ "database/sql"`,
			wantPath:  "database/sql",
			wantAlias: "_",
		},
		{
			name:      "long import path",
			line:      `"gitlab.mvk.com/go/vkgo/projects/health/steps/pkg/ntime"`,
			wantPath:  "gitlab.mvk.com/go/vkgo/projects/health/steps/pkg/ntime",
			wantAlias: "",
		},
		{
			name:      "long import path with alias",
			line:      `ntime "gitlab.mvk.com/go/vkgo/projects/health/steps/pkg/ntime"`,
			wantPath:  "gitlab.mvk.com/go/vkgo/projects/health/steps/pkg/ntime",
			wantAlias: "ntime",
		},
		{
			name:      "invalid line without quotes",
			line:      `fmt`,
			wantPath:  "",
			wantAlias: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotAlias := parseImportLine(tt.line)
			if gotPath != tt.wantPath {
				t.Errorf("parseImportLine() gotPath = %v, want %v", gotPath, tt.wantPath)
			}
			if gotAlias != tt.wantAlias {
				t.Errorf("parseImportLine() gotAlias = %v, want %v", gotAlias, tt.wantAlias)
			}
		})
	}
}

func TestStringHasPrefixAny(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		prefixes []string
		want     bool
	}{
		{
			name:     "matching prefix",
			s:        "github.com/myorg/myrepo",
			prefixes: []string{"github.com/myorg", "gitlab.com/myorg"},
			want:     true,
		},
		{
			name:     "no matching prefix",
			s:        "github.com/otherorg/repo",
			prefixes: []string{"github.com/myorg", "gitlab.com/myorg"},
			want:     false,
		},
		{
			name:     "empty prefixes",
			s:        "github.com/myorg/repo",
			prefixes: []string{},
			want:     false,
		},
		{
			name:     "empty string",
			s:        "",
			prefixes: []string{"github.com/myorg"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stringHasPrefixAny(tt.s, tt.prefixes); got != tt.want {
				t.Errorf("stringHasPrefixAny() = %v, want %v", got, tt.want)
			}
		})
	}
}
