# Template Usage

This directory is the working template for a lab-report project.

## Setup

1. Ensure `typst` is installed and available in your shell.
2. Open `report.typ` and fill in the lab metadata variables.
3. Put your solution code in `src/` and snippets for the report in `snippets/`.
4. Add screenshots to `img/lab/` when the report needs them. Use `node capture.js` to capture terminal output into PNGs.
5. Adjust the resulting PDF filename template in `.prepare.config` if needed.
6. Run `node prepare.js` to compile the report and create the submission bundle in `submission/`.

## Required Tools

- `typst` for compiling the report.
- `zip` on Linux and macOS for source archiving.
- PowerShell on Windows for source archiving.
- `charmbracelet/freeze` and ImageMagick's `magick` command for terminal screenshot capture.

## Commands

Compile the report and create the submission bundle:

```bash
node prepare.js
```

Reconfigure the output-name template before packaging:

```bash
node prepare.js --configure
```

Capture terminal output into a PNG:

```bash
node capture.js img/lab/session "typst compile report.typ report.pdf"
```

> [!IMPORTANT] 
> You probably want to configure freeze's user settings first with `freeze -i` 

## Output

`prepare.js` writes the compiled PDF and source archive into `submission/`.
The PDF and archive names are derived from the template in `.prepare.config`
unless you reconfigure it with `--configure`.
