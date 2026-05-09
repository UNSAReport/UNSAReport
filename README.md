# Lab Report CLI

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
- [VHS](https://github.com/charmbracelet/vhs): For terminal output capture.

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

### Updating Templates

Keep your project's template files synchronized with the latest version:

```bash
lab-report update
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

Leveraging VHS, you can capture terminal sessions directly into PNG files.

**Oneshot Mode**: Execute instructions directly from the command line.

```bash
lab-report capture --cwd ./src output.png "python script.py" "tape:Sleep 2s" "tape:Ctrl+D"
```

- Text arguments are typed into the terminal followed by `Enter`.
- Arguments prefixed with `tape:` are interpreted as raw VHS commands.
- If a `config.tape` file exists in the directory, it is automatically loaded.

**Script Mode**: Run an existing `.tape` file.

```bash
lab-report capture --tape report.tape
```

## Project Structure

### Single Lab
```text
.
├── labreport.json    # Project configuration
├── report.typ        # Main report file
├── lib.typ           # Template library
├── src/              # Source code directory
├── img/              # Image assets
└── submission/       # Generated PDF and ZIP
```

### Multi Lab
```text
.
├── labreport.json    # Project configuration
├── lib.typ           # Template library
├── img/              # Fixed image assets
├── l1/               # Lab 1 directory
│   ├── report.typ    # Lab 1 report file
│   ├── img/          # Lab 1 specific images
│   ├── src/          # Lab 1 source code
│   └── submission/   # Lab 1 generated PDF and ZIP
└── l2/
    ├── report.typ
    └── ...
```

## Configuration

The `labreport.json` file in your project root controls the behavior of the tool:

- `multiLab`: (boolean) Indicates if the project is a multi-lab setup.
- `capture`:
  - `tapeConfig`: (string) The VHS tape configuration file automatically loaded during capture. Defaults to `config.tape`.
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
