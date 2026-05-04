#!/usr/bin/env node

/**
 * Install the lab report template into a destination directory.
 *
 * This copies the template files that are meant to be edited in a project
 * workspace while keeping user-generated output folders out of the install.
 */

import {
  copyFileSync,
  existsSync,
  mkdirSync,
  readdirSync,
  statSync,
} from "node:fs";
import { createRequire } from "node:module";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const require = createRequire(import.meta.url);

function printHelp() {
  console.log(`
Lab Report Template Installer

Usage:
  node scripts/install.js [options]
  lab-report-install [options]

Options:
  --dest <path>   Destination directory (default: current working directory)
  -h, --help      Show this help message

Examples:
  lab-report-install --dest ~/projects/lab-01
  node scripts/install.js --dest .
`);
}

async function run() {
  const args = process.argv.slice(2);
  let destDir = process.cwd();

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg === "--dest") {
      if (i + 1 >= args.length) {
        throw new Error("Missing value for --dest");
      }

      destDir = resolve(args[i + 1]);
      i++;
    } else if (arg === "--help" || arg === "-h") {
      printHelp();
      return;
    } else {
      throw new Error(`Unknown option: ${arg}`);
    }
  }

  let sourceDir;
  try {
    const moduleDir = dirname(
      require.resolve("@christianmz565/lab-report/package.json"),
    );
    sourceDir = join(moduleDir, "template");
  } catch (_e) {
    sourceDir = join(__dirname, "..", "template");
  }

  if (!existsSync(sourceDir)) {
    throw new Error(`Template source not found at ${sourceDir}`);
  }

  if (!existsSync(destDir)) {
    mkdirSync(destDir, { recursive: true });
  }

  const filesToCopy = [
    "lib.typ",
    "report.typ",
    "prepare.js",
    "capture.js",
    "flake.nix",
    "bibliography.bib",
    "README.md",
    ".prepare.config",
  ];

  const dirsToCopy = ["img", "snippets", "src", "guide"];

  console.log(`Installing lab report template to: ${destDir}`);
  console.log("--------------------------------------------------");

  function copyFileWithLog(src, dst) {
    const displayPath = relative(destDir, dst);
    if (existsSync(src)) {
      if (!existsSync(dst)) {
        copyFileSync(src, dst);
        console.log(`Copied:  ${displayPath}`);
      } else {
        console.log(`Skipped: ${displayPath} (already exists)`);
      }
    }
  }

  function copyDirRecursive(src, dst) {
    if (!existsSync(dst)) {
      mkdirSync(dst, { recursive: true });
    }

    const files = readdirSync(src);
    for (const file of files) {
      const srcPath = join(src, file);
      const dstPath = join(dst, file);

      if (statSync(srcPath).isDirectory()) {
        copyDirRecursive(srcPath, dstPath);
      } else {
        copyFileWithLog(srcPath, dstPath);
      }
    }
  }

  for (const file of filesToCopy) {
    const src = join(sourceDir, file);
    const dst = join(destDir, file);
    copyFileWithLog(src, dst);
  }

  for (const dir of dirsToCopy) {
    const src = join(sourceDir, dir);
    const dst = join(destDir, dir);

    if (existsSync(src)) {
      copyDirRecursive(src, dst);
    }
  }

  console.log("--------------------------------------------------");
  console.log("Installation complete!");
  console.log("");
  console.log("Next steps:");
  console.log("1. Edit report.typ with your lab information");
  console.log("2. Place your code snippets in snippets/");
  console.log("3. Place your lab code in src/");
  console.log("4. Add screenshots to img/lab/ as needed");
  console.log("5. Run prepare.js in the template directory when ready");
}

run().catch((err) => {
  console.error(err.message ?? err);
  process.exit(1);
});
