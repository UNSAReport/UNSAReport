# Template Usage (Multi-lab)

This directory is the working template for a multi-lab repository.

## Setup

1. Ensure `typst` is installed and available in your shell (or use `nix develop`).
2. For each lab (e.g., `l1/`, `l2/`):
    - Open `report.typ` in the lab directory and fill in the lab metadata variables.
    - Put your solution code in `src/`, screenshots in `img/lab/` and (snippets)[#code-blocks-and-snippets] in `snippets/`. (within the lab directory).
3. Run `lab-report prepare <lab-dir>` (or `lab-report prepare` from within the lab directory) to compile the report and create the submission bundle.

## Typst Template Features

The shared `lib.typ` file provides several components:

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
The `#code-block` component reads files directly from your lab directory:
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

Compile the report and create the submission bundle for a specific lab (e.g., `l1`):
```bash
lab-report prepare l1
```

Capture terminal output into a PNG:
```bash
lab-report capture output.png "python l1/src/main.py" "tape:Ctrl+D"
```
