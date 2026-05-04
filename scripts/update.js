#!/usr/bin/env node

/**
 * Update an installed project from the packaged template files.
 *
 * The update flow compares tracked template files and asks before copying
 * over changes unless forced.
 */

import { execSync } from "node:child_process";
import { copyFileSync, existsSync, readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname, join, resolve } from "node:path";
import { createInterface } from "node:readline/promises";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const require = createRequire(import.meta.url);

function printHelp() {
  console.log(`
Lab Report Template Updater

Usage:
  node scripts/update.js [options]
  lab-report-update [options]

Options:
  --dest <path>   Destination directory (default: current working directory)
  -f, --force     Apply all updates without prompting
  -h, --help      Show this help message

Examples:
  lab-report-update --dest ~/projects/lab-01
  node scripts/update.js --force
`);
}

async function run() {
  const args = process.argv.slice(2);
  let destDir = process.cwd();
  let force = false;

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg === "--dest") {
      if (i + 1 >= args.length) {
        throw new Error("Missing value for --dest");
      }

      destDir = resolve(args[i + 1]);
      i++;
    } else if (arg === "--force" || arg === "-f") {
      force = true;
    } else if (arg === "--help" || arg === "-h") {
      printHelp();
      return;
    } else {
      throw new Error(`Unknown option: ${arg}`);
    }
  }

  let sourceDir;
  try {
    sourceDir = join(
      dirname(require.resolve("@christianmz565/lab-report/package.json")),
      "template",
    );
  } catch (_e) {
    sourceDir = join(__dirname, "..", "template");
  }

  const filesToCheck = [
    "lib.typ",
    "report.typ",
    "prepare.js",
    "capture.js",
    "flake.nix",
    "bibliography.bib",
    "README.md",
  ];

  const rl = createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  console.log(`Checking for updates in: ${destDir}\n`);

  let appliedCount = 0;
  let autoAcceptAll = force;

  try {
    for (const file of filesToCheck) {
      const src = join(sourceDir, file);
      const dst = join(destDir, file);

      if (!existsSync(src)) continue;

      const isNew = !existsSync(dst);
      const hasChanged =
        !isNew && readFileSync(src, "utf8") !== readFileSync(dst, "utf8");

      if (isNew || hasChanged) {
        if (isNew) {
          console.log(`\n[NEW FILE] ${file}`);
        } else {
          showSmartDiff(file, dst, src);
        }

        let shouldUpdate = false;

        if (autoAcceptAll) {
          shouldUpdate = true;
        } else {
          const answer = await rl.question(
            `Apply update to ${file}? (y)es / (n)o / (a)ll / (q)uit: `,
          );
          const choice = answer.toLowerCase();

          if (choice === "a") {
            autoAcceptAll = true;
            shouldUpdate = true;
          } else if (choice === "y") {
            shouldUpdate = true;
          } else if (choice === "q") {
            console.log("\nStopping update process.");
            break;
          }
        }

        if (shouldUpdate) {
          copyFileSync(src, dst);
          console.log(`✓ Updated: ${file}`);
          appliedCount++;
        } else {
          console.log(`Skipped: ${file}`);
        }
      }
    }
  } finally {
    rl.close();
  }

  console.log(`\nUpdate finished. ${appliedCount} files updated.`);
}

function showSmartDiff(fileName, oldPath, newPath) {
  console.log(`\n--- Diff for ${fileName} ---`);

  let hasDiff = false;
  try {
    execSync(process.platform === "win32" ? "where diff" : "command -v diff", {
      stdio: "ignore",
    });
    hasDiff = true;
  } catch (_e) {
    hasDiff = false;
  }

  if (hasDiff) {
    try {
      const colorFlag = process.stdout.isTTY ? "--color=always" : "";
      execSync(`diff -u ${colorFlag} "${oldPath}" "${newPath}"`, {
        stdio: "inherit",
      });
    } catch (e) {
      if (e.status !== 1) {
        console.log("  (Error running system diff)");
      }
    }
  } else {
    console.log("  [System 'diff' not found. Summary of changes:]");
    console.log(`  - Local version at: ${oldPath}`);
    console.log(`  + New template at: ${newPath}`);
    console.log("  (Install diff or Git to see inline diffs here)");
  }
  console.log("----------------------------\n");
}

run().catch((err) => {
  console.error(err.message ?? err);
  process.exit(1);
});
