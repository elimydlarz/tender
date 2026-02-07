#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const packageJsonPath = path.join(__dirname, "..", "package.json");
const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, "utf8"));
const packageVersion = packageJson.version;

const args = process.argv.slice(2);

if (args.includes("--help") || args.includes("-h")) {
  printHelp();
  process.exit(0);
}

if (args.includes("--version") || args.includes("-v")) {
  process.stdout.write(`${packageVersion}\n`);
  process.exit(0);
}

if (args.length > 0) {
  process.stderr.write(
    "tender is interactive-only. Run without arguments: npx --yes .\n"
  );
  process.exit(2);
}

const binPath = await resolveBinaryPath();
const exitCode = await execBinary(binPath);
process.exit(exitCode);

function printHelp() {
  process.stdout.write("tender\n\n");
  process.stdout.write("Interactive-only CLI launcher.\n\n");
  process.stdout.write("Usage:\n");
  process.stdout.write("  npx --yes .\n");
}

async function resolveBinaryPath() {
  if (process.env.TENDER_BINARY_PATH) {
    if (!fs.existsSync(process.env.TENDER_BINARY_PATH)) {
      throw new Error(`TENDER_BINARY_PATH not found: ${process.env.TENDER_BINARY_PATH}`);
    }
    return process.env.TENDER_BINARY_PATH;
  }

  const ext = process.platform === "win32" ? ".exe" : "";
  const candidates = [
    path.join(process.cwd(), "bin", `tender${ext}`),
    path.join(__dirname, "..", "bin", `tender${ext}`),
  ];
  for (const candidate of candidates) {
    if (fs.existsSync(candidate)) {
      return candidate;
    }
  }

  throw new Error(
    [
      "Local tender binary not found.",
      "Run `make build` from the repository root, then run `npx --yes .` again.",
      "Alternatively set TENDER_BINARY_PATH to a built tender binary.",
    ].join(" ")
  );
}

function execBinary(binPath) {
  return new Promise((resolve, reject) => {
    const child = spawn(binPath, [], { stdio: "inherit" });

    child.on("exit", (code, signal) => {
      if (signal) {
        reject(new Error(`tender terminated by signal ${signal}`));
        return;
      }
      resolve(code ?? 1);
    });

    child.on("error", reject);
  });
}
