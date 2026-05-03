#!/usr/bin/env node

/**
 * Compile the Typst report, resolve a configurable output name, and package
 * the source folder for submission.
 */

import { execSync } from "node:child_process";
import {
  copyFileSync,
  existsSync,
  mkdirSync,
  readFileSync,
  unlinkSync,
  writeFileSync,
} from "node:fs";
import { resolve as _resolve, dirname, join } from "node:path";
import { createInterface } from "node:readline/promises";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

function printHelp() {
  console.log(`
Prepare the report submission bundle

Usage:
  node prepare.js [options]

Options:
  --configure   Recreate the output-name template prompt
  -h, --help    Show this help message

Examples:
  node prepare.js
  node prepare.js --configure
`);
}

function compileTypst(generatedTitle) {
  console.log("Compiling typst report...");
  execSync(
    `typst compile --input title="${generatedTitle}" "report.typ" "report.pdf"`,
    { stdio: "inherit" },
  );
}

function archiveSrc(zipPath) {
  if (!existsSync("src")) {
    throw new Error("src directory not found.");
  }

  console.log(`Archiving src to ${zipPath}...`);
  if (process.platform === "win32") {
    execSync(
      `powershell Compress-Archive -Path "src\\*" -DestinationPath "${zipPath}" -Force`,
      { stdio: "inherit" },
    );
    return;
  }

  const absZipPath = _resolve(zipPath);
  execSync(`cd src && zip -q -r "${absZipPath}" .`, { stdio: "inherit" });
}

async function run() {
  const args = process.argv.slice(2);
  const reconfigure = args.includes("--configure");
  const showHelp = args.includes("--help") || args.includes("-h");
  const configFile = ".prepare.config";

  if (showHelp) {
    printHelp();
    return;
  }

  try {
    execSync("typst --version", { stdio: "ignore" });
  } catch {
    throw new Error("typst not found");
  }

  const varsDict = {};
  try {
    const output = execSync('typst query report.typ "<var_export>"', {
      encoding: "utf-8",
    });
    const data = JSON.parse(output);

    for (const item of data) {
      if (item.value?.name) {
        const val = Array.isArray(item.value.value)
          ? item.value.value.join("-")
          : item.value.value;
        varsDict[item.value.name] = val;
      }
    }
  } catch (_e) {
    throw new Error(
      "typst query failed. Make sure typst is installed and report.typ exists.",
    );
  }

  const rl = createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  const question = (query) => rl.question(query);

  let template = "";
  if (existsSync(configFile) && !reconfigure) {
    template = readFileSync(configFile, "utf-8").trim();
  }

  while (!template) {
    console.log("\nVariable configuration for report naming:");
    console.log("Available variables:");
    for (const [k, v] of Object.entries(varsDict)) {
      if (
        typeof v === "string" ||
        typeof v === "number" ||
        typeof v === "boolean"
      ) {
        console.log(`  {${k}}: ${v}`);
      }
    }
    console.log("\nExample: Informe_LAB_{lab_number}_{members_abbr_list}");

    let inputTemplate = await question(
      "Enter the name template (no extension): ",
    );

    inputTemplate = inputTemplate.trim() || "Informe_LAB_{lab_number}";

    let generatedName = "";
    try {
      generatedName = inputTemplate.replace(/{(\w+)}/g, (match, p1) => {
        return varsDict[p1] !== undefined ? varsDict[p1] : match;
      });
    } catch (_e) {
      generatedName = "Informe_Laboratorio";
    }

    console.log(`\nPreview: ${generatedName}.pdf`);
    const confirm = await question("Keep this template? (y/n): ");

    if (confirm.toLowerCase() === "y") {
      template = inputTemplate;
      writeFileSync(configFile, template);
      console.log(`Template saved to ${configFile}`);
    }
  }

  rl.close();

  const generatedName = template.replace(/{(\w+)}/g, (match, p1) => {
    return varsDict[p1] !== undefined ? varsDict[p1] : match;
  });

  compileTypst(generatedName);

  if (!existsSync("submission")) {
    mkdirSync("submission");
  }

  const reportFile = `${generatedName}.pdf`;
  const codeFile = `${generatedName}.zip`;

  copyFileSync("report.pdf", join("submission", reportFile));

  const zipPath = join("submission", codeFile);
  if (existsSync(zipPath)) {
    unlinkSync(zipPath);
  }

  archiveSrc(zipPath);

  console.log(`\nReport: ${reportFile}`);
  console.log(`Code:  ${codeFile}`);
}

run().catch((err) => {
  console.error(err.message ?? err);
  process.exit(1);
});
