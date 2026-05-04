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

Install the template into a project directory with a one-off command:

```bash
npx @christianmz565/lab-report install --dest /path/to/lab-project
```

You can also use pnpm:

```bash
pnpx @christianmz565/lab-report install --dest /path/to/lab-project
```

The installer copies the editable template files and assets into the target
directory without overwriting files that already exist.

If you are working on this package itself, install dependencies locally with
`npm install` and use `node scripts/install.js` for manual testing.

## Update

If the published template changes later, update an existing project with:

```bash
npx @christianmz565/lab-report update --dest /path/to/lab-project
```

```bash
pnpx @christianmz565/lab-report update --dest /path/to/lab-project
```

Use `--force` to apply all tracked file updates without prompting.
