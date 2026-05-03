#!/usr/bin/env node

/**
 * Capture terminal output, optionally replaying interactive inputs, and render
 * the result to a PNG screenshot.
 */

import { spawnSync } from "node:child_process";
import { existsSync, mkdirSync, unlinkSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import pty from "node-pty";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

function printHelp() {
  console.log(`
Capture Command Output

Usage:
  node capture.js <output-file> <command> [ms:input] [...]

Arguments:
  <output-file>    Base name for the generated .png file
  <command>        Command to execute in a shell
  [ms:input]       Optional delayed input values for interactive commands

Options:
  -h, --help       Show this help message

Examples:
  node capture.js img/lab/output "typst compile report.typ report.pdf"
  node capture.js img/lab/session "node" 1000:console.log('done')
`);
}

async function run() {
  const args = process.argv.slice(2);
  if (args.includes("--help") || args.includes("-h")) {
    printHelp();
    return;
  }

  if (args.length < 2) {
    printHelp();
    throw new Error("Missing required arguments");
  }

  const outputFile = args[0];
  const remainingArgs = args.slice(1);

  const esc = "\x1b";
  const reset = `${esc}[0m`;
  const green = `${esc}[38;5;114m`;
  const blue = `${esc}[38;5;111m`;
  const flamingo = `${esc}[38;5;217m`;

  if (!existsSync("capture_logs")) {
    mkdirSync("capture_logs");
  }

  const runId = new Date().toISOString().replace(/[:.]/g, "-");
  const logFile = join("capture_logs", `lastcmd_${runId}.log`);

  let tempOutput = "";

  let i = 0;

  while (i < remainingArgs.length) {
    const commandName = remainingArgs[i];
    const words = commandName.split(/\s+/);
    const firstWord = words[0];
    const restWords = words.slice(1).join(" ");

    let coloredCommand = `${blue}${firstWord}${reset}`;
    if (restWords) coloredCommand += ` ${flamingo}${restWords}${reset}`;

    tempOutput += `${green}❯ ${coloredCommand}\n`;

    const inputs = [];
    while (i + 1 < remainingArgs.length && /^\d+:/.test(remainingArgs[i + 1])) {
      i++;
      inputs.push(remainingArgs[i]);
    }

    const isWin = process.platform === "win32";
    const shell = isWin ? "cmd.exe" : "bash";
    const shellArgs = isWin ? ["/c", commandName] : ["-c", commandName];

    await new Promise((resolve) => {
      const ptyProcess = pty.spawn(shell, shellArgs, {
        name: "xterm-256color",
        cols: 100,
        rows: 30,
        cwd: process.cwd(),
        env: { ...process.env, FORCE_COLOR: "1" },
      });

      ptyProcess.onData((data) => {
        tempOutput += data.replace(/\r/g, "");
      });

      ptyProcess.onExit(() => {
        resolve();
      });

      if (inputs.length > 0) {
        (async () => {
          for (const item of inputs) {
            const [ms, ...textParts] = item.split(":");
            const text = textParts.join(":");

            await new Promise((r) => setTimeout(r, Number.parseInt(ms, 10)));

            ptyProcess.write(`${text}\r`);
          }

          await new Promise((r) => setTimeout(r, 200));

          if (isWin) {
            ptyProcess.write("\x1A");
          } else {
            ptyProcess.write("\x04");
          }
        })();
      }
    });

    i++;
  }

  writeFileSync(logFile, tempOutput);

  try {
    console.log(`Generating ${outputFile}.png...`);

    const freezeResult = spawnSync(
      "freeze",
      [
        "--width",
        "1000",
        "--output",
        `${outputFile}.svg`,
        "--language",
        "ansi",
        "-c",
        "user"
      ],
      {
        input: tempOutput,
        encoding: "utf-8",
      },
    );

    if (freezeResult.status === 0) {
      spawnSync("magick", [
        "-density",
        "300",
        "-background",
        "none",
        `${outputFile}.svg`,
        `${outputFile}.png`,
      ]);

      if (existsSync(`${outputFile}.svg`)) {
        unlinkSync(`${outputFile}.svg`);
      }

      console.log("Success!");
    } else {
      throw new Error("'freeze' command failed. Cannot generate image.");
    }
  } catch (e) {
    throw new Error(`Error generating capture image: ${e.message}`);
  }
}

run().catch((err) => {
  console.error(err.message ?? err);
  process.exit(1);
});
