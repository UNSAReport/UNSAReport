# EPIS Lab Report Template

This repository packages a Typst-based report template for EPIS/UNSA lab work.
It is meant to be installed into a project directory and used to compile the
report, capture terminal output when needed, and bundle the submission.

## Requirements

- Node.js 18 or newer.
- `typst` on `PATH`.
- A working `zip` command on Linux and macOS, or PowerShell on Windows for the
	submission archive step.
- `charmbracelet/freeze` and ImageMagick's `magick` command if you want terminal captures.

The bundled `flake.nix` provides a ready-made environment with the template's
native tooling if you prefer Nix.

## Install

Install the workspace dependencies first:

```bash
npm install
```

Then install the template into a project directory:

```bash
npx lab-report-install --dest /path/to/lab-project
```

The installer copies the editable template files and assets into the target
directory without overwriting files that already exist.

## Update

If the published template changes later, update an existing project with:

```bash
npx lab-report-update --dest /path/to/lab-project
```

Use `--force` to apply all tracked file updates without prompting.
