#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const [command, ...rest] = process.argv.slice(2);

const commandMap = {
  install: "install.js",
  update: "update.js",
};

function printHelp() {
  console.log(`
Lab Report Template CLI

Usage:
  lab-report-template <command> [options]

Commands:
  install   Install the template files into a destination
  update    Update an existing template installation

Run with "--help" after a command for its options.

Examples:
  lab-report-template install --dest ~/projects/lab-01
  lab-report-template update --force
`);
}

if (!command) {
  printHelp();
  process.exit(0);
}

if (!(command in commandMap)) {
  console.error(`Unknown command: ${command}`);
  printHelp();
  process.exit(1);
}

const scriptPath = join(__dirname, commandMap[command]);
const result = spawnSync(process.execPath, [scriptPath, ...rest], {
  stdio: "inherit",
});

if (result.error) {
  console.error(result.error.message ?? result.error);
  process.exit(1);
}

process.exit(result.status ?? 1);
