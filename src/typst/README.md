# Template Usage

This directory is the working template for a lab-report project.

## Setup

1. Ensure `typst` is installed and available in your shell (or use `nix develop`).
2. Open `report.typ` and fill in the lab metadata variables.
3. Put your solution code in `src/` and snippets for the report in `snippets/`.
4. Add screenshots to `img/lab/` when the report needs them.
5. Run `lab-report prepare` to compile the report and create the submission bundle in `submission/`.

## Required Tools

- `typst` for compiling the report.
- `freeze` (charmbracelet/freeze) and ImageMagick's `magick` for terminal screenshot capture.

## Commands

Compile the report and create the submission bundle:

```bash
lab-report prepare
```

Reconfigure the output-name template before packaging:

```bash
lab-report prepare --configure
```

Capture terminal output into a PNG:

```bash
lab-report capture img/lab/session "typst compile report.typ report.pdf"
```

> You probably want to configure freeze's user settings first with `freeze -i`.

## Output

`lab-report prepare` writes the compiled PDF and source archive into `submission/`.
The PDF and archive names are derived from the template in `.prepare.config` unless you reconfigure it with `--configure`.
