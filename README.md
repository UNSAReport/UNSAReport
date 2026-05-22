# Lab Report CLI

> [!WARNING]
> This tool is particularly volatile as it is still being adapted to the actual needs to make work easier. It is very likely that some features are broken, don't exist, get removed randomly, or require manual intervention to work. Your best bet is to install the version in the dev branch, create an empty project and copy the latest labreport.json from it.

A command-line interface designed to automate and manage laboratory reports for the UNSA Software Engineering career. This tool streamlines the process of scaffolding projects, updating templates, capturing terminal output, and preparing final submissions.

## Template Features

The included Typst template provides specialized components for academic reports:

- **UNSA/EPIS Compliance**: Standardized page headers and layout according to the "Escuela Profesional de Ingeniería de Sistemas" requirements.
- **Metadata-driven Reports**: Define variables like `course_name`, `lab_title`, and `members` in your report to automatically populate the information tables.
- **Lab Sections**: Use `#lab-section(title: "...", body: [...])` to create formatted sections with header bars.
- **Advanced Code Blocks**: The `#code-block` component allows including source files directly with syntax highlighting and customizable snippets.
- **Named Snippets**: Extract specific portions of your code by wrapping them with `// START-SNIPPET,name` and `// END-SNIPPET` markers in your source files.

## Requirements

The CLI lazily validates external dependencies at runtime:

- [Typst](https://typst.app/): For report compilation and metadata extraction.
- [Freeze](https://github.com/charmbracelet/freeze): For terminal output capture.
- [ImageMagick](https://imagemagick.org/): For SVG to PNG conversion.

A `flake.nix` is provided for users who prefer a pre-configured Nix environment.

## Installation

### Using Go

```bash
go install github.com/christianmz565/lab-report/cmd/lab-report@latest
```

### Using Nix

```bash
nix run github:christianmz565/lab-report -- --help
```

## Usage

### Initializing a Project

To install the laboratory template in the current directory:

```bash
lab-report install
```

For a repository containing multiple laboratory sessions (e.g., `l1/`, `l2/`):

```bash
lab-report install --multi
```

To add a new session to an existing multi-lab repository:

```bash
lab-report install --session l3
```

### Updating Templates

Keep your project's template files synchronized with the latest version:

```bash
# Update the current single-lab project or ALL registered sessions in a multi-lab project
lab-report update

# Update a specific session in a multi-lab project
lab-report update l1
```

Use the `--force` flag to apply all updates without interactive prompts.

### Preparing Submissions

The `prepare` command compiles the report and archives the source code into the output folder (defaults to `submission/`). This will generate a PDF file with the compiled report and a ZIP file containing the source code. It uses metadata defined in your report file (defaults to `report.typ`) to name the files according to a configurable template.

If a `.git` directory is detected in the project root, the tool automatically filters out files in your source directory (defaults to `src/`) based on your `.gitignore` rules before creating the ZIP archive.

**Single-lab project:**

```bash
lab-report prepare
```

**Multi-lab project:**

```bash
# From the project root, specifying the lab directory
lab-report prepare l1

# Or from within the lab directory itself
cd l1
lab-report prepare
```

Use `--configure` to re-trigger the naming template setup prompt.

### Capturing Terminal Output

Leveraging Freeze and ImageMagick, you can capture terminal sessions directly into PNG files.

The `capture` command executes terminal instructions directly and captures the resulting output. It uses a clean terminal environment and supports custom prompt formatting.

```bash
lab-report capture --cwd ./src output.png "python script.py" "w:2s"
```

- Text arguments are typed into the terminal followed by `Enter`.
- Arguments prefixed with `w:` are interpreted as a wait/sleep:
  - `w:<duration>` (e.g., `w:2s`, `w:500ms`)
- Arguments prefixed with `r:` write the raw text after it without pressing Enter.
- Arguments prefixed with `c:` send a Ctrl + <key> combination (e.g., `c:c` for Ctrl+C).
- Arguments prefixed with `k:` send a specific control key (e.g., `k:enter`, `k:tab`, `k:backspace`, `k:escape`, `k:esc`).

Logs of the terminal output (including ANSI colors) are automatically saved in the `capture_logs/` directory as `.log` files.

## Project Structure

### Single Lab
```text
.
├── labreport.json    # Project configuration
├── report.typ        # Main report file
├── lib.typ           # Template library
├── functions.typ     # Useful functions for config var generation
├── README.md         # Summary file with instructions
├── flake.nix         # Nix flake for development environment
├── bibliography.bib  # Bibliography file for references
├── guide/            # Lab guide and instructions
├── src/              # Source code directory
├── snippets/         # Code snippets for the report directory
├── img/              # Image assets
└── submission/       # Generated PDF and ZIP
```

### Multi Lab
```text
.
├── labreport.json         # Project configuration
├── lib.typ                # Template library
├── functions.typ          # Useful functions for config var generation
├── README.md              # Summary file with instructions
├── flake.nix              # Nix flake for development environment
├── img/fixed/             # Fixed image assets
├── l1/                    # Lab 1 directory
│   ├── report.typ         # Lab 1 report file
│   ├── bibliography.bib   # Lab 1 bibliography file for references
│   ├── guide/             # Lab 1 guide and instructions
│   ├── snippets/          # Lab 1 code snippets for the report directory
│   ├── img/lab/           # Lab 1 specific images
│   ├── src/               # Lab 1 source code
│   └── submission/        # Lab 1 generated PDF and ZIP
└── l2/
    ├── report.typ
    └── ...
```

## Configuration

The `labreport.json` file in your project root controls the behavior of the tool:

- `multiLab`: (boolean) Indicates if the project is a multi-lab setup.
- `sessions`: (array of strings) List of registered session directories (e.g., `["l1", "l2"]`) in a multi-lab setup. Managed automatically by `install --session`.
- `capture`:
  - `columns`: (integer) The width of the terminal in characters (default: `120`).
  - `freezeFlags`: (array of strings) Additional flags to pass to `freeze` during capture.
  - `prompt`: (string) The prompt character to use (default: `❯ `).
  - `colors`: (object) ANSI color codes for syntax highlighting during capture:
    - `prompt`: Color for the prompt character.
    - `command`: Color for the first word of a command.
    - `args`: Color for the rest of the command arguments.
    - `reset`: ANSI code to reset formatting (usually `0`).
- `prepare`:
  - `input`:
    - `srcDir`: (string) The directory containing your source code. Defaults to `src`.
    - `reportFile`: (string) The main Typst report filename. Defaults to `report.typ`.
  - `output`:
    - `submissionDir`: (string) The directory where the compiled report and zip bundle will be saved. Defaults to `submission`.
    - `fileTemplate`: (string) Naming template for the generated PDF and ZIP files.
    - `reportWord`: (string) Word you want to use for "Report" in filenames.
    - `codeWord`: (string) Word you want to use for "Source Code" in filenames.

These customizable paths allow you to adapt the template structure to your needs.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
