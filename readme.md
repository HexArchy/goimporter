# goimporter - Universal Go Import Organizer

`goimporter` is a tool that automatically organizes and groups import statements in Go files according to configurable rules. It can be used as a standalone command-line tool or integrated with your editor.

## Features

- Organizes imports into logical groups:
  1. Standard library packages
  2. External dependencies
  3. Organization-specific common packages
  4. Domain-specific packages
  5. Project-specific `/pkg` packages (before internal packages)
  6. Project-specific `/internal` packages
- Customizable import prefix configurations
- Works with any repository structure through configuration
- Detects and skips generated files
- Provides dry-run mode to preview changes
- Recursive directory processing
- Can exclude mock files

## Installation

### Standard Installation

```bash
# Clone the repository
git clone https://github.com/HexArchy/goimporter.git
cd goimporter

# Build and install to your Go bin directory
make install-gobin

# Or install system-wide (requires sudo)
make install-global
```

### VK-Specific Installation

```bash
# Clone the repository
git clone https://github.com/HexArchy/goimporter.git
cd goimporter

# Install with VK-specific configuration
make install-vk
```

This will:
1. Install the tool to your Go bin directory
2. Create a VK-specific configuration file
3. Add a convenient `vkgoimporter` alias to your shell profile

## Usage

### Basic Usage

```bash
# Format a single file
goimporter path/to/file.go

# Format all Go files in current directory
goimporter

# Format files recursively
goimporter -r

# Format files in a specific directory
goimporter -dir=/path/to/directory

# Dry run (don't make changes, just show what would be done)
goimporter -d
```

### Using with Custom Repository Structure

```bash
goimporter -org "github.com/myorg" -repo "github.com/myorg/myrepo" -common-prefix "github.com/myorg/myrepo/pkg"
```

### Using a Configuration File

```bash
# Create a configuration file
cat > ~/.config/goimporter/config.json <<EOF
{
  "org_prefix": "github.com/myorg",
  "repo_prefix": "github.com/myorg/myrepo",
  "common_prefix": "github.com/myorg/myrepo/pkg",
  "domain_prefix": "github.com/myorg/myrepo/domain/pkg",
  "projects_template": "github.com/myorg/myrepo/projects/%s",
  "additional_common_prefixes": [
    "github.com/myorg/common-lib"
  ]
}
EOF

# Use the configuration file
goimporter -config ~/.config/goimporter/config.json
```

### VK-Specific Usage

After installing with `make install-vk`, you can use the convenient alias:

```bash
# Format a single file with VK-specific settings
vkgoimporter path/to/file.go

# Format recursively with VK-specific settings
vkgoimporter -r
```

## Configuration Options

| Flag             | Description                               | Default                               |
| ---------------- | ----------------------------------------- | ------------------------------------- |
| `-dir`           | Directory to process                      | Current directory                     |
| `-r`             | Process files recursively                 | false                                 |
| `-d`             | Dry run mode                              | false                                 |
| `-exclude-mock`  | Exclude mock files                        | true                                  |
| `-config`        | Path to config file (JSON)                | ""                                    |
| `-org`           | Organization prefix                       | "github.com/myorg"                    |
| `-repo`          | Repository prefix                         | "github.com/myorg/myrepo"             |
| `-common-prefix` | Common packages prefix                    | "github.com/myorg/myrepo/pkg"         |
| `-domain-prefix` | Domain-specific packages prefix           | "github.com/myorg/myrepo/domain/pkg"  |
| `-projects-tpl`  | Projects template                         | "github.com/myorg/myrepo/projects/%s" |
| `-pkgs`          | Custom package prefixes (comma-separated) | ""                                    |

## Integration with Editors

### VSCode

1. Create a format script:

```bash
#!/bin/bash
# ~/.config/goimporter/format.sh
goimporter -config ~/.config/goimporter/config.json "$@"
```

2. Make it executable:

```bash
chmod +x ~/.config/goimporter/format.sh
```

3. Configure VSCode in settings.json:

```json
{
  "[go]": {
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
      "source.organizeImports": true
    },
  },
  "go.formatTool": "custom",
  "go.formatFlags": [
    "~/.config/goimporter/format.sh"
  ]
}
```

### GoLand/IntelliJ IDEA

Configure as an external tool:

1. Go to Settings → Tools → External Tools
2. Add a new tool:
   - Name: goimporter
   - Program: path to goimporter binary
   - Arguments: `-config ~/.config/goimporter/config.json $FilePath$`
   - Working directory: `$ProjectFileDir$`
3. Assign a keyboard shortcut in Settings → Keymap

## Development

### Project Structure

```
goimporter/
├── cmd/
│   └── goimporter/     # Main application entry point
├── config/             # Configuration handling
├── formatter/          # Import formatting logic
├── model/              # Core data types
├── Makefile            # Build and installation tasks
└── install_vk.sh       # VK-specific installation script
```

### Building from Source

```bash
# Clone the repository
git clone https://github.com/HexArchy/goimporter.git
cd goimporter

# Build
make build

# Run tests
make test
```

### Quick Installation for VK Colleagues

```bash
bash -c "$(curl -sSL https://github.com/HexArchy/goimporter/raw/master/install_vk.sh)"
```

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.