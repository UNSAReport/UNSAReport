# Lab Report CLI

This repository provides:

- A Typst-based report template (fetched during install/update).
- A cross-platform `lab-report` CLI that can scaffold/update projects, compile submissions, and capture terminal output.

## Requirements

The CLI validates these external lazily at runtime:

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

or

```bash
cd /path/to/lab/01
lab-report prepare
```

Use `--configure` to re-run the PDF naming template prompt.

## Capture terminal output

Thanks to the incredible [charmbracelet/vhs](https://github.com/charmbracelet/vhs) project, terminal output capture is fully scriptable.

You can use the **oneshot mode** to generate a capture directly from arguments:

```bash
lab-report capture --cwd my-project output.png "python script.py" "tape:Ctrl+D"
```

In oneshot mode:
- The `capture` command takes a result image path (`output.png`) followed by terminal instructions.
- `--cwd` automatically navigates to a directory before executing your instructions.
- Text without a prefix is typed automatically (`Type "<text>"; Enter; Sleep 2`).
- Text starting with `tape:` is interpreted directly as VHS instructions (e.g. `tape:Ctrl+D`, `tape:Sleep 5s`). You can escape it with `\tape:`.
- If a `config.tape` file is present in the working directory, it will automatically be sourced.

Alternatively, you can run a `.tape` script file directly using the `--tape` flag:

```bash
lab-report capture --tape template.tape
```
