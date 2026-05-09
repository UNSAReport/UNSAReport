# Template Usage

This directory is the working template for a lab-report project.

## Setup

1. Ensure `typst` is installed and available in your shell (or use `nix develop`).
2. Open `report.typ` (or your configured report filename) and fill in the lab metadata variables.
3. Put your solution code in `src/` (or your configured source directory), screenshots in `img/lab/` and (snippets)[#code-blocks-and-snippets] in `snippets/`.
4. Run `lab-report prepare` to compile the report and create the submission bundle in `submission/` (or your configured output directory).

## Project Configuration

You can customize the project structure in `labreport.json`:

- `prepare.input.reportFile`: Set this if you rename `report.typ`.
- `prepare.input.srcDir`: Set this if your code is in a directory other than `src/`.
- `prepare.output.submissionDir`: Change where the final files are generated.
- `capture.tapeConfig`: Change the default VHS configuration file.

## Typst Template Features

The `lib.typ` file provides several components to streamline report creation:

### Metadata and Layout
The top level `#lab-report()[]` rule initializes the UNSA/EPIS compliant layout. It automatically extracts variables exported via `<var_export>`:
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
- `vhs` (charmbracelet/vhs) for terminal screenshot capture.

## Commands

Compile the report and create the submission bundle:
```bash
lab-report prepare
```

Capture terminal output into a PNG (you'll also get an .ascii file with the raw output under capture_logs when using oneshot mode):
```bash
lab-report capture output.png "python src/main.py" "tape:Ctrl+D"
```
