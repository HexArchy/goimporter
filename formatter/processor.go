package formatter

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"goimporter/config"
)

// ProcessFile organizes imports in a single Go file.
func ProcessFile(filename string, cfg *config.Config) error {
	code, err := os.ReadFile(filename)
	if err != nil {
		return errors.Wrap(err, "reading file")
	}

	// Check if this is a generated file - if so, skip it.
	if IsGeneratedFile(code) {
		fmt.Printf("Skipping generated file: %s\n", filename)
		return nil
	}

	// Get prefixes for this file.
	prefixes := GetImportPrefixes(filename, cfg.Repo)
	if len(cfg.PkgPrefixes) > 0 {
		prefixes = cfg.PkgPrefixes
	}

	// Collect all imports from the file.
	allImports, err := CollectImports(code)
	if err != nil {
		return errors.Wrap(err, "collecting imports")
	}

	// If no imports were found, nothing to do.
	if len(allImports) == 0 {
		return nil
	}

	// Group imports and remove duplicates.
	groups := GroupImports(allImports, prefixes, cfg.Repo)

	// Generate the new file content.
	newContent, err := RewriteFile(code, groups)
	if err != nil {
		return errors.Wrap(err, "rewriting file")
	}

	// Skip writing if content didn't change.
	if bytes.Equal(code, newContent) {
		return nil
	}

	// Only write changes if not in dry run mode.
	if !cfg.DryRun {
		err := os.WriteFile(filename, newContent, 0o644)
		if err != nil {
			return errors.Wrap(err, "writing file")
		}
		fmt.Printf("Processed: %s\n", filename)
	} else {
		fmt.Printf("Would process: %s\n", filename)
	}

	return nil
}

// ProcessGoFiles processes all Go files in a directory or recursively.
func ProcessGoFiles(cfg *config.Config) error {
	if cfg.Recursive {
		err := filepath.WalkDir(cfg.Dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.IsDir() && strings.HasSuffix(path, ".go") {
				if cfg.ExcludeMock && strings.Contains(path, "mock") {
					return nil
				}

				err := ProcessFile(path, cfg)
				if err != nil {
					fmt.Printf("Error processing %s: %v\n", path, err)
				}
			}
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "walking directory")
		}
		return nil
	}

	// Process only go files in the specified directory.
	entries, err := os.ReadDir(cfg.Dir)
	if err != nil {
		return errors.Wrap(err, "reading directory")
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			if cfg.ExcludeMock && strings.Contains(entry.Name(), "mock") {
				continue
			}

			path := filepath.Join(cfg.Dir, entry.Name())
			err := ProcessFile(path, cfg)
			if err != nil {
				fmt.Printf("Error processing %s: %v\n", path, err)
			}
		}
	}

	return nil
}
