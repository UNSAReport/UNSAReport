# Template Usage (Multi-lab)

This directory is the working template for a multi-lab repository.

## Setup

1. Ensure `typst` is installed and available in your shell (or use `nix develop`).
2. For each lab (e.g., `l1/`, `l2/`):
    - Open `report.typ` in the lab directory and fill in the lab metadata variables.
    - Put your solution code in `src/` and snippets for the report in `snippets/` (within the lab directory).
    - Add screenshots to `img/lab/` when the report needs them.
3. Run `lab-report prepare <lab-dir>` (or just `lab-report prepare` from within the lab directory) to compile the report and create the submission bundle.

## Required Tools

- `typst` for compiling the report.
- `vhs` (charmbracelet/vhs) for terminal screenshot capture.

## Commands

Compile the report and create the submission bundle for a specific lab (e.g., `l1`):

```bash
lab-report prepare l1
```

> You can also run `lab-report prepare` directly from within the `l1/` directory.

Reconfigure the output-name template before packaging:

```bash
lab-report prepare --configure l1
```

Capture terminal output into a PNG via a `.tape` file script:

```bash
lab-report capture template.tape
```

### VHS Tape Scripting

A `.tape` file consists of commands that control a virtual terminal.

- `Screenshot <path>.png`: Save the current frame as a PNG.
- `Type "<string>"`: Emulate typing characters.
- `Enter`, `Backspace`, `Up`, `Down`, `Left`, `Right`, `Tab`, `Space`, `Escape`: Emulate pressing the respective keys.
- `Ctrl+<char>`, `Alt+<char>`: Emulate pressing control/alt combined with a character.
- `Sleep <time>`: Wait for a certain amount of time (e.g. `Sleep 1s`).
- `Hide`: Pause capturing.
- `Show`: Resume capturing.

#### Settings

Settings must be at the top of the file (or in `config.tape`):
- `Set Width <number>`: Set terminal width.
- `Set Height <number>`: Set terminal height.
- `Set FontSize <number>`: Set font size.
- `Set FontFamily "<font>"`: Set font family.
- `Set Theme "<name>"`: Set the terminal theme.

## Output

`lab-report prepare` writes the compiled PDF and source archive into the `submission/` folder inside the lab directory (e.g., `l1/submission/`).
The PDF and archive names are derived from the template in `labreport.json` unless you reconfigure it with `--configure`.
