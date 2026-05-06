#!/usr/bin/env node

/**
 * Bundle template runtime scripts from src/runtime into template using ncc.
 * Copy typst assets from src/typst into template.
 */

import {
  chmodSync,
  copyFileSync,
  existsSync,
  mkdirSync,
  readdirSync,
  statSync,
  writeFileSync,
} from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";
import ncc from "@vercel/ncc";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const templateDir = join(__dirname, "..", "template");
if (!existsSync(templateDir)) {
  mkdirSync(templateDir, { recursive: true });
}

const entries = [
  {
    input: join(__dirname, "..", "src", "runtime", "capture.cjs"),
    output: join(templateDir, "capture.js"),
  },
  {
    input: join(__dirname, "..", "src", "runtime", "prepare.cjs"),
    output: join(templateDir, "prepare.js"),
  },
];

for (const entry of entries) {
  const { code } = await ncc(entry.input, {
    minify: true,
  });

  writeFileSync(entry.output, code, "utf-8");
  chmodSync(entry.output, 0o755);
  console.log(`Bundled ${entry.output}`);
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
      copyFileSync(srcPath, dstPath);
      console.log(`Copied ${relative(join(__dirname, ".."), dstPath)}`);
    }
  }
}

const typstDir = join(__dirname, "..", "src", "typst");
if (existsSync(typstDir)) {
  copyDirRecursive(typstDir, templateDir);
}
