#!/usr/bin/env node

/**
 * Lab Report CLI Entrypoint
 *
 * Routes commands to the appropriate installation or update scripts.
 */

import { spawnSync } from "node:child_process";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const COMMAND_MAP = {
  install: "install.js",
  update: "update.js",
};

const HELP_TEXT = `
Lab Report Template CLI

Usage:
  lab-report <command> [options]

Commands:
  install   Install the template files into a destination
  update    Update an existing template installation

Run with "--help" after a command for its options.

Examples:
  lab-report install --dest ~/projects/lab-01
  lab-report update --force
`;

async function run() {
  const args = process.argv.slice(2);
  const command = args[0];
  const restArgs = args.slice(1);

  if (!command || command === "-h" || command === "--help") {
    console.log(HELP_TEXT.trim());
    return;
  }

  if (!(command in COMMAND_MAP)) {
    console.error(`Error: Unknown command "${command}"\n`);
    console.log(HELP_TEXT.trim());
    process.exit(1);
  }

  const scriptPath = join(__dirname, COMMAND_MAP[command]);
  const result = spawnSync(process.execPath, [scriptPath, ...restArgs], {
    stdio: "inherit",
  });

  if (result.error) {
    throw result.error;
  }

  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}

run().catch((err) => {
  console.error(err.message ?? err);
  process.exit(1);
});
