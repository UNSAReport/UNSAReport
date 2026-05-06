#!/usr/bin/env node

/**
 * Update an installed project from the packaged template files.
 */

import { execSync } from "node:child_process";
import {
  copyFileSync,
  existsSync,
  readdirSync,
  readFileSync,
  statSync,
  mkdirSync,
  writeFileSync,
} from "node:fs";
import { createRequire } from "node:module";
import { dirname, join, resolve, relative } from "node:path";
import { createInterface } from "node:readline/promises";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const require = createRequire(import.meta.url);

const COMMON_MANIFEST = [
  { src: "lib.typ", dest: "lib.typ", autoUpdate: false },
  { src: "flake.nix", dest: "flake.nix", autoUpdate: false },
  { src: "bibliography.bib", dest: "bibliography.bib", autoUpdate: false },
  { src: "README.md", dest: "README.md", autoUpdate: false },
  { src: ".prepare.config", dest: ".prepare.config", autoUpdate: false },
  { src: "prepare.js", dest: "prepare.js", autoUpdate: true },
  { src: "capture.js", dest: "capture.js", autoUpdate: true },
];

const HELP_TEXT = `
Lab Report Template Updater

Usage:
  node scripts/update.js [options]
  lab-report-update [options]

Options:
  --dest <path>   Destination directory (default: current working directory)
  --multi         Force multi-lab update mode
  -f, --force     Apply all updates without prompting
  -h, --help      Show this help message

Examples:
  lab-report-update --dest ~/projects/lab-01
  node scripts/update.js --force --multi
`;

class Updater {
  constructor(sourceDir, destDir, force, multiLab) {
    this.sourceDir = sourceDir;
    this.destDir = destDir;
    this.autoAcceptAll = force;
    this.multiLab = multiLab;
    this.appliedCount = 0;
    this.rl = createInterface({ input: process.stdin, output: process.stdout });
  }

  getFilesRecursively(dir) {
    const files = [];
    const read = (currentDir) => {
      for (const entry of readdirSync(currentDir, { withFileTypes: true })) {
        const fullPath = join(currentDir, entry.name);
        if (entry.isDirectory()) read(fullPath);
        else files.push(fullPath);
      }
    };
    read(dir);
    return files;
  }

  buildManifest() {
    const configPath = join(this.destDir, "labreport.json");
    let isMulti = this.multiLab;

    if (existsSync(configPath)) {
      try {
        const config = JSON.parse(readFileSync(configPath, "utf-8"));
        isMulti = isMulti || config.multiLab;
      } catch (e) {
        console.warn(`Error reading labreport.json: ${e.message}`);
      }
    } else {
      const defaultConfig = { multiLab: isMulti };
      writeFileSync(configPath, JSON.stringify(defaultConfig, null, 2));
      console.warn(
        `Warning: labreport.json not found in ${this.destDir}. Created default config.`,
      );
      console.warn("Please verify the configuration and run the command again.");
      process.exit(1);
    }

    console.log(`Detected ${isMulti ? "multi-lab" : "single-lab"} setup.`);
    const manifest = [...COMMON_MANIFEST];

    if (isMulti) {
      const readmeIndex = manifest.findIndex((m) => m.dest === "README.md");
      if (readmeIndex !== -1) {
        manifest[readmeIndex] = {
          ...manifest[readmeIndex],
          src: "multi/README.md",
        };
      }

      const subDirs = readdirSync(this.destDir).filter((f) => {
        try {
          return (
            statSync(join(this.destDir, f)).isDirectory() &&
            !f.startsWith(".") &&
            f !== "node_modules"
          );
        } catch {
          return false;
        }
      });

      for (const dir of subDirs) {
        if (existsSync(join(this.destDir, dir, "report.typ"))) {
          manifest.push({
            src: "multi/report.typ",
            dest: `${dir}/report.typ`,
            autoUpdate: false,
          });
        }
      }

      const fixedImgDir = join(this.sourceDir, "img", "fixed");
      if (existsSync(fixedImgDir)) {
        const files = this.getFilesRecursively(fixedImgDir);
        for (const file of files) {
          const relPath = relative(fixedImgDir, file).replace(/\\/g, "/");
          manifest.push({
            src: `img/fixed/${relPath}`,
            dest: `img/fixed/${relPath}`,
            autoUpdate: false,
          });
        }
      }
    } else {
      manifest.push({
        src: "report.typ",
        dest: "report.typ",
        autoUpdate: false,
      });
    }

    return manifest;
  }

  async processUpdates() {
    const manifest = this.buildManifest();
    console.log(`Checking for updates in: ${this.destDir}\n`);

    try {
      for (const item of manifest) {
        const srcPath = join(this.sourceDir, item.src);
        const dstPath = join(this.destDir, item.dest);
        const displayLabel = item.dest;

        if (!existsSync(srcPath)) continue;

        const isNew = !existsSync(dstPath);
        const hasChanged =
          !isNew &&
          !readFileSync(srcPath).equals(readFileSync(dstPath));

        if (isNew || hasChanged) {
          await this.handleFileUpdate(
            item,
            srcPath,
            dstPath,
            displayLabel,
            isNew,
          );
        }
      }
    } finally {
      this.rl.close();
    }

    console.log(`\nUpdate finished. ${this.appliedCount} files updated.`);
  }

  async handleFileUpdate(item, srcPath, dstPath, label, isNew) {
    const applyUpdate = () => {
      const dstDir = dirname(dstPath);
      if (!existsSync(dstDir)) {
        mkdirSync(dstDir, { recursive: true });
      }
      copyFileSync(srcPath, dstPath);
      this.appliedCount++;
    };

    if (item.autoUpdate || this.autoAcceptAll) {
      applyUpdate();
      const tag = item.autoUpdate ? "[AUTO-UPDATE]" : "[FORCED]";
      console.log(`✓ Updated ${tag}: ${label}`);
      return;
    }

    if (isNew) {
      console.log(`\n[NEW FILE] ${label}`);
    } else {
      this.showSmartDiff(label, dstPath, srcPath);
    }

    let shouldUpdate = false;
    while (true) {
      const answer = await this.rl.question(
        `Apply update to ${label}? (y)es / (n)o / (a)ll / (q)uit: `,
      );
      const choice = answer.toLowerCase().trim();

      if (choice === "a") {
        this.autoAcceptAll = true;
        shouldUpdate = true;
        break;
      } else if (choice === "y") {
        shouldUpdate = true;
        break;
      } else if (choice === "n") {
        break;
      } else if (choice === "q") {
        console.log("\nStopping update process.");
        this.rl.close();
        process.exit(0);
      }
    }

    if (shouldUpdate) {
      applyUpdate();
      console.log(`✓ Updated: ${label}`);
    } else {
      console.log(`Skipped: ${label}`);
    }
  }

  showSmartDiff(fileName, oldPath, newPath) {
    if (fileName.match(/\.(png|jpe?g|gif|svg|webp|ico)$/i)) {
      console.log(`\n--- Diff for ${fileName} ---`);
      console.log("  [Binary image file changed. Cannot display inline diff.]");
      console.log(`  - Local version at: ${oldPath}`);
      console.log(`  + New template at: ${newPath}`);
      console.log(`${"-".repeat(30)}\n`);
      return;
    }

    console.log(`\n--- Diff for ${fileName} ---`);
    let hasDiff = false;

    try {
      execSync(
        process.platform === "win32" ? "where diff" : "command -v diff",
        { stdio: "ignore" },
      );
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
        if (e.status !== 1) console.log("  (Error running system diff)");
      }
    } else {
      console.log("  [System 'diff' not found. Summary of changes:]");
      console.log(`  - Local version at: ${oldPath}`);
      console.log(`  + New template at: ${newPath}`);
      console.log("  (Install diff or Git to see inline diffs here)");
    }
    console.log(`${"-".repeat(30)}\n`);
  }
}

function parseArgs() {
  const args = process.argv.slice(2);
  const config = { destDir: process.cwd(), force: false, multiLab: false, help: false };

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg === "--dest") {
      if (i + 1 >= args.length) throw new Error("Missing value for --dest");
      config.destDir = resolve(args[++i]);
    } else if (arg === "--force" || arg === "-f") {
      config.force = true;
    } else if (arg === "--multi") {
      config.multiLab = true;
    } else if (arg === "--help" || arg === "-h") {
      config.help = true;
    } else {
      throw new Error(`Unknown option: ${arg}`);
    }
  }
  return config;
}

function resolveSourceDir() {
  try {
    return join(
      dirname(require.resolve("@christianmz565/lab-report/package.json")),
      "template",
    );
  } catch (_e) {
    return join(__dirname, "..", "..", "template");
  }
}

async function run() {
  const { destDir, force, multiLab, help } = parseArgs();

  if (help) {
    console.log(HELP_TEXT.trim());
    return;
  }

  const sourceDir = resolveSourceDir();

  if (!existsSync(sourceDir)) {
    throw new Error(`Template source not found at ${sourceDir}`);
  }

  const updater = new Updater(sourceDir, destDir, force, multiLab);
  await updater.processUpdates();
}

run().catch((err) => {
  console.error(err.message ?? err);
  process.exit(1);
});