/**
 * Compile the Typst report, resolve a configurable output name, and package
 * the source folder for submission.
 */

const { execSync } = require("node:child_process");
const {
  copyFileSync,
  existsSync,
  mkdirSync,
  readFileSync,
  unlinkSync,
  writeFileSync,
} = require("node:fs");
const { resolve: resolvePath, join } = require("node:path");
const { createInterface } = require("node:readline/promises");

function printHelp() {
  console.log(`
Prepare the report submission bundle

Usage:
  node prepare.js [options] [lab-dir]

Options:
  --configure   Recreate the output-name template prompt
  -h, --help    Show this help message

Arguments:
  [lab-dir]     Optional subdirectory containing the lab (e.g. l1)

Examples:
  node prepare.js
  node prepare.js l1
  node prepare.js --configure l1
`);
}

function compileTypst(generatedTitle, reportPath, reportPdf, isMulti) {
  console.log("Compiling typst report...");
  const rootArg = isMulti ? "--root ." : "";
  execSync(
    `typst compile ${rootArg} --input title="${generatedTitle}" "${reportPath}" "${reportPdf}"`,
    { stdio: "inherit" },
  );
}

function archiveSrc(zipPath, srcDir) {
  if (!existsSync(srcDir)) {
    console.warn(
      `Warning: ${srcDir} directory not found. Skipping zip generation.`,
    );
    return;
  }

  console.log(`Archiving ${srcDir} to ${zipPath}...`);
  if (process.platform === "win32") {
    execSync(
      `powershell Compress-Archive -Path "${srcDir}\\*" -DestinationPath "${zipPath}" -Force`,
      { stdio: "inherit" },
    );
    return;
  }

  const absZipPath = resolvePath(zipPath);
  execSync(`cd "${srcDir}" && zip -q -r "${absZipPath}" .`, {
    stdio: "inherit",
  });
}

async function run() {
  const args = process.argv.slice(2);
  const reconfigure = args.includes("--configure");
  const showHelp = args.includes("--help") || args.includes("-h");

  const positionalArgs = args.filter((a) => !a.startsWith("-"));
  const labDirArg = positionalArgs[0] || "";

  const configPath = "labreport.json";
  let isMulti = false;

  if (existsSync(configPath)) {
    try {
      const config = JSON.parse(readFileSync(configPath, "utf-8"));
      isMulti = !!config.multiLab;
    } catch (e) {
      console.warn(`Error reading labreport.json: ${e.message}`);
    }
  } else {
    const defaultConfig = { multiLab: !!labDirArg };
    writeFileSync(configPath, JSON.stringify(defaultConfig, null, 2));
    console.warn(
      "Warning: labreport.json not found. Created default config.",
    );
    console.warn("Please verify the configuration and run the command again.");
    process.exit(1);
  }

  if (isMulti && !labDirArg) {
    throw new Error(
      "Multi-lab mode detected. Please specify a lab directory (e.g., node prepare.js l1).",
    );
  }

  const labDir = labDirArg;
  const reportPath = isMulti ? join(labDir, "report.typ") : "report.typ";
  const reportPdf = isMulti ? join(labDir, "report.pdf") : "report.pdf";
  const srcDir = isMulti ? join(labDir, "src") : "src";

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

  if (!existsSync(reportPath)) {
    throw new Error(`Report file not found: ${reportPath}`);
  }

  const varsDict = {};
  try {
    const rootArg = isMulti ? "--root ." : "";
    const output = execSync(
      `typst query ${rootArg} "${reportPath}" "<var_export>"`,
      {
        encoding: "utf-8",
      },
    );
    const data = JSON.parse(output);

    for (const item of data) {
      if (item.value?.name) {
        const val = Array.isArray(item.value.value)
          ? item.value.value.join("-")
          : item.value.value;
        varsDict[item.value.name] = val;
      }
    }
  } catch (e) {
    throw new Error(
      `typst query failed. Make sure typst is installed and ${reportPath} exists.\n${e.message}`,
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

  compileTypst(generatedName, reportPath, reportPdf, isMulti);

  const submissionDir = isMulti ? join(labDir, "submission") : "submission";
  if (!existsSync(submissionDir)) {
    mkdirSync(submissionDir, { recursive: true });
  }

  const reportFile = `${generatedName}.pdf`;
  const codeFile = `${generatedName}.zip`;

  copyFileSync(reportPdf, join(submissionDir, reportFile));

  const zipPath = join(submissionDir, codeFile);
  if (existsSync(zipPath)) {
    unlinkSync(zipPath);
  }

  archiveSrc(zipPath, srcDir);

  console.log(`\nReport: ${join(submissionDir, reportFile)}`);
  console.log(`Code:  ${join(submissionDir, codeFile)}`);
}

run().catch((err) => {
  console.error(err.message ?? err);
  process.exit(1);
});
