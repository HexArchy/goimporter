package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"goimporter/entities"

	"github.com/pkg/errors"
)

// Config holds the configuration for the import processor.
type Config struct {
	Dir         string
	Recursive   bool
	DryRun      bool
	ExcludeMock bool
	PkgPrefixes []string
	ConfigPath  string
	Repo        *entities.RepoConfig
}

// DefaultRepoConfig creates a default repository configuration.
func DefaultRepoConfig() *entities.RepoConfig {
	return &entities.RepoConfig{
		OrgPrefix:        "github.com/myorg",
		RepoPrefix:       "github.com/myorg/myrepo",
		CommonPrefix:     "github.com/myorg/myrepo/pkg",
		DomainPrefix:     "github.com/myorg/myrepo/projects/domain/pkg",
		ProjectsTemplate: "github.com/myorg/myrepo/projects/domain/%s",
	}
}

// ParseFlags parses command line arguments into a Config.
func ParseFlags() *Config {
	cfg := &Config{
		Repo: DefaultRepoConfig(),
	}

	flag.StringVar(&cfg.Dir, "dir", ".", "Directory to process")
	flag.BoolVar(&cfg.Recursive, "r", false, "Process files recursively")
	flag.BoolVar(&cfg.DryRun, "d", false, "Don't write changes, just report")
	flag.BoolVar(&cfg.ExcludeMock, "exclude-mock", true, "Exclude mock files")
	flag.StringVar(&cfg.ConfigPath, "config", "", "Path to config file (JSON)")

	// Repository configuration flags.
	flag.StringVar(&cfg.Repo.OrgPrefix, "org", cfg.Repo.OrgPrefix, "Organization prefix")
	flag.StringVar(&cfg.Repo.RepoPrefix, "repo", cfg.Repo.RepoPrefix, "Repository prefix")
	flag.StringVar(&cfg.Repo.CommonPrefix, "common-prefix", cfg.Repo.CommonPrefix, "Common packages prefix")
	flag.StringVar(&cfg.Repo.DomainPrefix, "domain-prefix", cfg.Repo.DomainPrefix, "Domain-specific packages prefix")
	flag.StringVar(&cfg.Repo.ProjectsTemplate, "projects-tpl", cfg.Repo.ProjectsTemplate, "Projects template")

	customPkgs := flag.String("pkgs", "", "Custom package prefixes (comma-separated)")

	flag.Parse()

	// Load config file if specified.
	if cfg.ConfigPath != "" {
		err := cfg.loadConfigFile()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		}
	}

	// Command-line flags override config file.
	if *customPkgs != "" {
		cfg.PkgPrefixes = strings.Split(*customPkgs, ",")
	}

	return cfg
}

// loadConfigFile loads configuration from a JSON file.
func (c *Config) loadConfigFile() error {
	data, err := os.ReadFile(c.ConfigPath)
	if err != nil {
		return errors.Wrap(err, "reading config file")
	}

	var repo entities.RepoConfig
	err = json.Unmarshal(data, &repo)
	if err != nil {
		return errors.Wrap(err, "parsing config file")
	}

	// Update config with values from file.
	c.Repo = &repo
	return nil
}
