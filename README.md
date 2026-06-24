# UNSAReport CLI

> [!WARNING]
> This tool is under active development. Features may change, break, or require manual intervention. Use the `dev` branch for the latest state and copy the `unsareport.json` from a fresh project.

A command-line tool for automating lab report creation in the UNSA Software Engineering program. It scaffolds Typst-based projects, manages versioned templates and reusable components, captures terminal output, and compiles everything into submission-ready deliverables.

## Templates

Templates are maintained in a separate repository: [UNSAReport/templates](https://github.com/UNSAReport/templates).

| Template | Description |
|----------|-------------|
| `lab` | Single laboratory report |
| `multi-lab` | Multiple laboratory sessions in one repository |

Each template provides Typst components for academic reports: metadata-driven layouts, lab sections, code blocks with syntax highlighting, and named snippets. See the [templates README](https://github.com/UNSAReport/templates) for details on creating custom templates.

## Requirements

The CLI lazily validates external dependencies at runtime:

| Tool | Purpose |
|------|---------|
| [Typst](https://typst.app/) | Report compilation and metadata extraction |
| [Freeze](https://github.com/charmbracelet/freeze) | Terminal output capture |
| [ImageMagick](https://imagemagick.org/) | SVG to PNG conversion |

A `flake.nix` is provided for users who prefer a pre-configured Nix environment.

## Installation

### Using Go

```bash
go install github.com/UNSAReport/UNSAReport/cmd/unsarep@latest
```

### Using Nix

```bash
nix run github:UNSAReport/UNSAReport -- --help
```

## Commands

### `unsarep install` — Scaffold a project

Downloads template files and initializes an `unsareport.json` configuration.

```bash
# Interactive template picker
unsarep install

# Install a specific template
unsarep install lab
unsarep install multi-lab

# Install a specific version
unsarep install lab@1.0.0
unsarep install lab@^1.0.0

# Install from a local directory
unsarep install --local ./my-templates

# Install into a specific directory
unsarep install lab --dest ./my-reports
```

For multi-lab repositories, add new sessions with:

```bash
unsarep install multi-lab --session l3
```

### `unsarep update` — Synchronize with upstream templates

Compares local files against the latest template version. Shows a diff for each changed file and prompts before applying, unless `--force` is used. A backup is created before changes and can be restored with `--rollback`.

```bash
# Interactive update (per-file diff + prompt)
unsarep update

# Apply all updates without prompting
unsarep update --force

# Update a specific session in a multi-lab project
unsarep update l1

# Update from a local directory
unsarep update --local ./my-templates

# Rollback the last update
unsarep update --rollback
```

### `unsarep prepare` — Compile and package submissions

Compiles the Typst report to PDF and archives the source code into a ZIP file in the output directory (default: `submission/`). Uses metadata from `report.typ` to name files according to a configurable template. When a `.git` directory is present, the archive respects `.gitignore` rules.

```bash
# Single-lab project
unsarep prepare

# Multi-lab project (from root)
unsarep prepare l1

# Multi-lab project (from within the lab directory)
cd l1
unsarep prepare

# Re-trigger the naming template setup
unsarep prepare --configure
```

### `unsarep capture` — Capture terminal output

Executes terminal instructions in a virtual terminal and renders the output to a PNG using Freeze and ImageMagick.

```bash
# Basic capture
unsarep capture output.png "ls -la" "cat README.md"

# With custom working directory and delays
unsarep capture --cwd ./src result.png "python" "print('hello')" "w:1s"

# Using control keys
unsarep capture output.png "python" "print('hello')" "k:enter" "c:d"
```

**Instruction syntax:**

| Prefix | Behavior | Example |
|--------|----------|---------|
| *(none)* | Type text and press Enter | `"ls -la"` |
| `w:` | Wait/sleep for a duration | `"w:2s"`, `"w:500ms"` |
| `r:` | Write raw text without Enter | `"r:some text"` |
| `c:` | Send Ctrl+key combination | `"c:c"` (Ctrl+C) |
| `k:` | Send a control key | `"k:enter"`, `"k:tab"`, `"k:escape"` |

Logs of the terminal output (including ANSI colors) are saved in the `capture_logs/` directory.

### `unsarep component` — Manage Typst components

Add, remove, list, and update reusable Typst components from the [UNSAReport/components](https://github.com/UNSAReport/components) registry.

```bash
# List available components
unsarep component list

# Add a component
unsarep component add code-block
unsarep component add code-block@^1.0.0

# Remove a component
unsarep component remove code-block

# Update all components
unsarep component update

# Update a specific component
unsarep component update code-block
```

### `unsarep completion` — Shell completions

```bash
unsarep completion bash   # Bash
unsarep completion zsh    # Zsh
unsarep completion fish   # Fish
unsarep completion powershell  # PowerShell
```

## Configuration

The `unsareport.json` file in your project root controls CLI behavior. A JSON schema is published alongside each release for editor autocompletion.

```jsonc
{
  "$schema": "https://raw.githubusercontent.com/UNSAReport/UNSAReport/v1.0/schemas/unsareport.schema.json",
  "template": "lab",
  "templateVersion": "1.2.0",
  "mode": "single",            // "single" or "multi"
  "sessions": ["l1", "l2"],    // only in multi mode
  "components": {
    "code-block": {
      "version": "1.0.0",
      "installed_at": "2026-01-15T10:30:00Z"
    }
  },
  "capture": {
    "columns": 120,
    "freezeFlags": [],
    "prompt": "❯ ",
    "colors": {
      "prompt": "32",
      "command": "36",
      "args": "33",
      "reset": "0"
    }
  },
  "prepare": {
    "input": {
      "srcDir": "src",
      "reportFile": "report.typ"
    },
    "output": {
      "submissionDir": "submission",
      "fileTemplate": "{output_type}_{lab_number}",
      "reportWord": "Informe",
      "codeWord": "Código Fuente"
    }
  }
}
```

### Lockfile

An `unsareport.lock` file is automatically maintained to track installed template and component versions with integrity hashes. Do not edit this file manually.

### Environment Variables

All environment variables are prefixed with `UNSAREP_`:

| Variable | Description |
|----------|-------------|
| `UNSAREP_DEST` | Default destination directory |
| `UNSAREP_SESSION` | Default session for multi-lab projects |
| `UNSAREP_LOCAL` | Default local template source |
| `UNSAREP_FREEZE_FLAGS` | Default flags for Freeze |

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
