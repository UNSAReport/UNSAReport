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
- `vhs` (charmbracelet/vhs) for terminal screenshot capture.

## Commands

Compile the report and create the submission bundle:

```bash
lab-report prepare
```

Reconfigure the output-name template before packaging:

```bash
lab-report prepare --configure
```

Capture terminal output into a PNG.

**Oneshot mode:**
```bash
lab-report capture output.png "python script.py" "tape:Ctrl+D"
```

**Using a tape file:**
```bash
lab-report capture --tape template.tape
```

### VHS Tape Scripting

In oneshot mode, text without a prefix is automatically typed (`Type "<text>"; Enter; Sleep 2`). You can use the `tape:` prefix to use VHS directives directly.

A `.tape` file (or `tape:` prefix) consists of commands that control a virtual terminal.

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

`lab-report prepare` writes the compiled PDF and source archive into `submission/`.
The PDF and archive names are derived from the template in `labreport.json` unless you reconfigure it with `--configure`.
