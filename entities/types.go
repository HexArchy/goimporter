package entities

// Import represents a single import statement.
type Import struct {
	Alias string
	Path  string
}

// ImportGroups organizes imports into logical groups.
type ImportGroups struct {
	Stdlib          []Import // Standard library packages.
	External        []Import // External dependencies.
	OrgCommon       []Import // Common organization packages.
	DomainCommon    []Import // Domain packages.
	RepoOther       []Import // Other repository packages.
	ProjectPkg      []Import // Project-specific pkg packages.
	ProjectInternal []Import // Project-specific internal packages.
}

// RepoConfig holds organization and repository configuration.
type RepoConfig struct {
	// Organization prefix (e.g. "github.com/myorg").
	OrgPrefix string `json:"org_prefix"`

	// Repository prefix (e.g. "github.com/myorg/myrepo").
	RepoPrefix string `json:"repo_prefix"`

	// Common packages prefix (e.g. "github.com/myorg/myrepo/pkg").
	CommonPrefix string `json:"common_prefix"`

	// Domain-specific packages prefix (e.g. "github.com/myorg/myrepo/projects/domain/pkg").
	DomainPrefix string `json:"domain_prefix"`

	// Projects template for project-specific imports (e.g. "github.com/myorg/myrepo/projects/domain/%s").
	ProjectsTemplate string `json:"projects_template"`

	// Additional special repository prefixes that should be grouped with common packages.
	AdditionalCommonPrefixes []string `json:"additional_common_prefixes"`
}
