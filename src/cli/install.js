#!/usr/bin/env node

/**
 * Install the lab report template into a destination directory.
 */

import {
  copyFileSync,
  existsSync,
  mkdirSync,
  readdirSync,
  statSync,
  writeFileSync,
} from "node:fs";
import { createRequire } from "node:module";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const require = createRequire(import.meta.url);

const COMMON_FILES = [
  "lib.typ",
  "prepare.js",
  "capture.js",
  "flake.nix",
  "bibliography.bib",
  "README.md",
  ".prepare.config",
].map((file) => ({ type: "file", src: file, dest: file }));

const SINGLE_LAB_MANIFEST = [
  ...COMMON_FILES,
  { type: "file", src: "report.typ", dest: "report.typ" },
  { type: "dir", src: "img/lab", dest: "img/lab" },
  { type: "dir", src: "snippets", dest: "snippets" },
  { type: "dir", src: "src", dest: "src" },
  { type: "dir", src: "guide", dest: "guide" },
];

const MULTI_LAB_MANIFEST = [
  ...COMMON_FILES.filter((f) => f.src !== "README.md"),
  { type: "file", src: "multi/README.md", dest: "README.md" },
  { type: "dir", src: "img/fixed", dest: "img/fixed" },
  { type: "file", src: "multi/report.typ", dest: "l1/report.typ" },
  { type: "dir", src: "img/lab", dest: "l1/img/lab" },
  { type: "dir", src: "snippets", dest: "l1/snippets" },
  { type: "dir", src: "src", dest: "l1/src" },
  { type: "dir", src: "guide", dest: "l1/guide" },
];

const HELP_TEXT = `
Lab Report Template Installer

Usage:
  node scripts/install.js [options]
  lab-report install [options]

Options:
  --dest <path>   Destination directory (default: current working directory)
  --multi         Install as a multi-lab repository (global tools at root, lab files in l1/)
  -h, --help      Show this help message

Examples:
  lab-report install --dest ~/projects/lab-01
  lab-report install --dest ~/projects/labs --multi
`;

const getNextSteps = (multiLab) =>
  multiLab
    ? [
        "1. Edit l1/report.typ with your lab information",
        "2. Place your code snippets in l1/snippets/",
        "3. Place your lab code in l1/src/",
        "4. Add screenshots to l1/img/lab/ as needed",
        "5. Run 'node prepare.js l1' in the root directory when ready",
        "6. Duplicate l1 folder for subsequent labs (e.g., l2, l3)",
      ]
    : [
        "1. Edit report.typ with your lab information",
        "2. Place your code snippets in snippets/",
        "3. Place your lab code in src/",
        "4. Add screenshots to img/lab/ as needed",
        "5. Run prepare.js in the template directory when ready",
      ];

class Installer {
  constructor(sourceDir, destDir) {
    this.sourceDir = sourceDir;
    this.destDir = destDir;
  }

  processManifest(manifest) {
    for (const item of manifest) {
      const srcPath = join(this.sourceDir, item.src);
      const dstPath = join(this.destDir, item.dest);

      if (!existsSync(srcPath)) continue;

      if (item.type === "file") {
        this.copyFile(srcPath, dstPath);
      } else if (item.type === "dir") {
        this.copyDir(srcPath, dstPath);
      }
    }
  }

  copyFile(src, dst) {
    const displayPath = relative(this.destDir, dst) || ".";
    const dstDir = dirname(dst);

    if (!existsSync(dstDir)) mkdirSync(dstDir, { recursive: true });

    if (!existsSync(dst)) {
      copyFileSync(src, dst);
      console.log(`Copied:  ${displayPath}`);
    } else {
      console.log(`Skipped: ${displayPath} (already exists)`);
    }
  }

  copyDir(src, dst) {
    if (!existsSync(dst)) mkdirSync(dst, { recursive: true });

    const files = readdirSync(src);
    for (const file of files) {
      const srcPath = join(src, file);
      const dstPath = join(dst, file);

      if (statSync(srcPath).isDirectory()) {
        this.copyDir(srcPath, dstPath); // recurse
      } else {
        this.copyFile(srcPath, dstPath);
      }
    }
  }
}

function parseArgs() {
  const args = process.argv.slice(2);
  const config = { destDir: process.cwd(), multiLab: false, help: false };

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg === "--dest") {
      if (i + 1 >= args.length) throw new Error("Missing value for --dest");
      config.destDir = resolve(args[++i]);
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
    const moduleDir = dirname(
      require.resolve("@christianmz565/lab-report/package.json"),
    );
    return join(moduleDir, "template");
  } catch (_e) {
    return join(__dirname, "..", "..", "template");
  }
}

async function run() {
  const { destDir, multiLab, help } = parseArgs();

  if (help) {
    console.log(HELP_TEXT.trim());
    return;
  }

  const sourceDir = resolveSourceDir();

  if (!existsSync(sourceDir)) {
    throw new Error(`Template source not found at ${sourceDir}`);
  }

  if (!existsSync(destDir)) {
    mkdirSync(destDir, { recursive: true });
  }

  console.log(`Installing lab report template to: ${destDir}`);
  if (multiLab) console.log(`Mode: Multi-lab (--multi flag active)`);
  console.log("-".repeat(50));

  const installer = new Installer(sourceDir, destDir);
  const manifest = multiLab ? MULTI_LAB_MANIFEST : SINGLE_LAB_MANIFEST;

  installer.processManifest(manifest);

  const labReportConfig = { multiLab };
  writeFileSync(
    join(destDir, "labreport.json"),
    JSON.stringify(labReportConfig, null, 2),
  );
  console.log(`Created: labreport.json (Mode: ${multiLab ? "multi" : "single"})`);

  console.log("-".repeat(50));
  console.log("Installation complete!\n");
  console.log("Next steps:");
  getNextSteps(multiLab).forEach((step) => {
    console.log(step);
  });
}

run().catch((err) => {
  console.error(err.message ?? err);
  process.exit(1);
});
