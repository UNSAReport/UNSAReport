# Template Usage

This directory is the working template for a unsa-report project.

## Setup

1. Ensure `typst` is installed and available in your shell (or use `nix develop`).
2. Open `report.typ` (or your configured report filename) and fill in the lab metadata variables.
3. Put your solution code in `src/` (or your configured source directory), screenshots in `img/lab/` and (snippets)[#code-blocks-and-snippets] in `snippets/`.
4. Run `unsarep prepare` to compile the report and create the submission bundle in `submission/` (or your configured output directory).

## Project Configuration

You can customize the project structure in `unsareport.json`:

- `prepare.input.reportFile`: Set this if you rename `report.typ`.
- `prepare.input.srcDir`: Set this if your code is in a directory other than `src/`.
- `prepare.output.submissionDir`: Change where the final files are generated.
- `capture.freezeFlags`: List of additional flags for terminal capture (e.g., `["--theme", "dracula"]`).
- `capture.columns`: The width of the terminal in characters (default: `120`).
- `capture.prompt`: The prompt character to use (e.g., `❯ `).
- `capture.colors`: ANSI color codes for terminal elements (`prompt`, `command`, `args`, `reset`).

## Typst Template Features

The `lib.typ` file provides several components to streamline report creation:

### Metadata and Layout
The top level `#unsa-report()[]` rule initializes the UNSA/EPIS compliant layout. It automatically extracts variables exported via `<var_export>`:
- `course_name`, `lab_title`, `lab_number`, `instructor_name`, `members`.
- Optional: `year`, `presentation_date`, `sem_code`, `presentation_hour`.
- Any other variables can be defined and used freely for the submission filename template.

### Lab Sections
Use the `#lab-section` component to create sections with a header bar:
```typst
#lab-section("Ejercicios")[
  Contenido de la sección...
])
```

### Code Blocks and Snippets
The `#code-block` component reads files directly:
```typst
#code-block("src/main.py", lang: "python")
```

To include only a specific part of a file, use named snippets:
1. In your source file:
   ```python
   # START-SNIPPET,my_logic
   print("This is the snippet")
   # END-SNIPPET
   ```
2. In your report:
   ```typst
   #code-block("src/main.py", snippet: "my_logic", lang: "python", prefix: "#")
   ```

You are strongly advised to configure language and prefix for snippet comments using [elembic's](https://typst.app/universe/package/elembic/) show rules instead.

```typst
#show: e.set_(code-block, lang: "python")
#show: e.set_(code-block, prefix: "#")
```

You are also strongly advised to follow the snippets/ and src/ conventions. All the contents of src/ are automatically included in the submission bundle.

## Required Tools

- `typst` for compiling the report.
- `freeze` (charmbracelet/freeze) and ImageMagick's `magick` for terminal screenshot capture.

## Commands

Update the template files to the latest version:
```bash
unsarep update
```

Compile the report and create the submission bundle:
```bash
unsarep prepare
```

Capture terminal output into a PNG (you'll also get a .log file with the raw output under `capture_logs/`):
```bash
unsarep capture output.png "python src/main.py"
```

- Text arguments are typed into the terminal followed by `Enter`.
- Arguments prefixed with `w:` are interpreted as a wait/sleep:
  - `w:<duration>` (e.g., `w:2s`, `w:500ms`)
- Arguments prefixed with `r:` write the raw text after it without pressing Enter.
- Arguments prefixed with `c:` send a Ctrl + <key> combination (e.g., `c:c` for Ctrl+C).
- Arguments prefixed with `k:` send a specific control key (e.g., `k:enter`, `k:tab`, `k:backspace`, `k:escape`, `k:esc`).
