---
name: lab-report
description: Manage and automate the creation of laboratory reports using the lab-report CLI and Typst template. Use this skill when the user wants to initialize a new lab report project, write content for a report, capture terminal output for inclusion in a report, or prepare the final submission (PDF and ZIP).
---

# Lab Report Skill

This skill provides guidance and automation for managing laboratory reports for the UNSA Software Engineering career using the `lab-report` CLI tool and its associated Typst template.

## Core Workflows

### 1. Initializing and Updating Projects
Use the `lab-report` CLI to scaffold your project.

- **Single Lab:** `lab-report install`
- **Multi-Lab:** `lab-report install --multi`
- **Add Session:** `lab-report install --session <name>`
- **Update Template:** `lab-report update [--force]`

### 2. Writing the Report (Typst)
The report is written in `report.typ` (or equivalent). It uses several custom components defined in `lib.typ`.

#### Metadata Definition
Always define the required metadata at the top of the report:
```typst
#define("course_name", "Ingeniería de Software")
#define("lab_title", "Título de la Práctica")
#define("lab_number", "01")
#define("instructor_name", "Nombre del Docente")
#define("members", (
  "Lastname1 Lastname1 Name1 Name1",
  "Lastname2 Lastname2 Name2 Name2",
))
```

#### Specialized Components
- **`#lab-report()[ ... ]`**: The main layout wrapper. All report content must be inside it.
- **`#lab-section(title, body)`**: Creates a formatted section with a header bar.
  - Typical sections: "RESULTADOS Y PRUEBAS", "CUESTIONARIO", "CONCLUSIONES", "REFERENCIAS".
- **`#code-block(file: "path", snippet: "name", lang: "ext")`**: Displays source code.
  - Use `snippet` to include only parts of a file (see Named Snippets below).
  - Use `lang` for syntax highlighting.

#### Named Snippets
In your source files, wrap code segments with markers:
```python
// START-SNIPPET,my-logic
print("This is my logic")
// END-SNIPPET
```
Then reference it in Typst:
```typst
#code-block(file: "src/main.py", snippet: "my-logic", lang: "python")
```

### 3. Capturing Terminal Output
Use `lab-report capture` to generate PNGs of terminal output for your report.

**Command structure:**
```bash
lab-report capture --cwd ./src output.png "command to run" "w:2s" "next command"
```
- `"w:<duration>"`: Wait (e.g., `w:1s`).
- `"c:<key>"`: Control key (e.g., `c:c` for Ctrl+C).
- `"k:<key>"`: Special key (e.g., `k:enter`, `k:esc`).

Captured images should be placed in the `img/` (or `img/lab/`) directory and included in the Typst report:
```typst
#figure(
  image("img/output.png", width: 80%),
  caption: [Resultado de la ejecución del script.],
)
```

### 4. Preparing Submissions
When the report is ready, generate the final PDF and source code ZIP.

```bash
lab-report prepare
```
- This generates files in the `submission/` directory.
- It uses the metadata in `report.typ` to name the files.

## Project Structure
Adhere to the standard structure:
- `report.typ`: Main report.
- `src/`: Source code.
- `img/`: Images and captures.
- `labreport.json`: CLI configuration.
- `submission/`: Generated output.

## Guidelines
- **Language**: Reports for UNSA are typically in Spanish.
- **Verification**: Always run `lab-report prepare` to ensure the report compiles without errors before finishing.
- **Captures**: Use `capture` for terminal output instead of manual screenshots to ensure a consistent and clean look.
- **Snippets**: Prefer `code-block` with `snippet` over including entire files to keep the report focused.
