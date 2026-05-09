# Lab Report CLI

This repository provides:

- A Typst-based report template (fetched during install/update).
- A cross-platform `lab-report` CLI that can scaffold/update projects, compile submissions, and capture terminal output.

## Requirements

The CLI validates these external tools on startup:

- `typst`
- `vhs` (charmbracelet/vhs)

If you prefer Nix, the included `flake.nix` provides a ready-made environment.

## Install the CLI

### Go

```bash
go install github.com/christianmz565/lab-report/cmd/lab-report@latest
```

### Nix

```bash
nix run github:christianmz565/lab-report -- --help
```

## Install a template into a project

```bash
lab-report install --dest /path/to/lab-project
```

Multi-lab layout (global files at repo root, labs under `l1/`, `l2/`, ...):

```bash
lab-report install --dest /path/to/labs --multi
```

## Update an existing project

```bash
lab-report update --dest /path/to/lab-project
```

Use `--force` to apply all updates without prompting.

## Prepare a submission

Single-lab:

```bash
cd /path/to/lab-project
lab-report prepare
```

Multi-lab:

```bash
cd /path/to/labs
lab-report prepare l1
```

Use `--configure` to re-run the PDF naming template prompt.

## Capture terminal output

Thanks to the incredible [charmbracelet/vhs](https://github.com/charmbracelet/vhs) project, terminal output capture is fully scriptable using `.tape` files!

```bash
lab-report capture template.tape
```

The tool will automatically create a default `config.tape` and `template.tape` if they are missing.
