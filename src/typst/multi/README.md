# Template Usage (Multi-lab)

This directory is the working template for a multi-lab repository.

## Setup

1. Ensure `typst` is installed and available in your shell.
2. For each lab (e.g., `l1/`, `l2/`):
    - Open `report.typ` in the lab directory and fill in the lab metadata variables.
    - Put your solution code in `src/` and snippets for the report in `snippets/` (within the lab directory).
    - Add screenshots to `img/lab/` when the report needs them.
3. Adjust the resulting PDF filename template in `.prepare.config` if needed.
4. Run `lab-report prepare <lab-dir>` to compile the report and create the submission bundle.

## Required Tools

- `typst` for compiling the report.
- `freeze` (charmbracelet/freeze) and ImageMagick's `magick` command for terminal screenshot capture.

## Commands

Compile the report and create the submission bundle for a specific lab (e.g., `l1`):

```bash
lab-report prepare l1
```

Reconfigure the output-name template before packaging:

```bash
lab-report prepare --configure l1
```

Capture terminal output into a PNG for a specific lab:

```bash
lab-report capture l1/img/lab/session "typst compile l1/report.typ l1/report.pdf"
```

> [!IMPORTANT]
> You probably want to configure freeze's user settings first with `freeze -i`

## Output

`lab-report prepare` writes the compiled PDF and source archive into the `submission/` folder inside the lab directory (e.g., `l1/submission/`).
The PDF and archive names are derived from the template in `.prepare.config`
unless you reconfigure it with `--configure`.
