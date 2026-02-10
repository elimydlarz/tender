#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import os from "node:os";
import { spawn } from "node:child_process";
import { Readable } from "node:stream";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const packageJsonPath = path.join(__dirname, "..", "package.json");
const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, "utf8"));
const packageVersion = packageJson.version;
const releaseBaseUrl =
  process.env.TENDER_RELEASE_BASE_URL ||
  "https://github.com/elimydlarz/tender/releases/download";

const args = process.argv.slice(2);

if (args.includes("--help") || args.includes("-h")) {
  printHelp();
  process.exit(0);
}

if (args.includes("--version") || args.includes("-v")) {
  process.stdout.write(`${packageVersion}\n`);
  process.exit(0);
}

const binPath = await resolveBinaryPath();
const exitCode = await execBinary(binPath, args);
process.exit(exitCode);

function printHelp() {
  process.stdout.write("tender\n\n");
  process.stdout.write("Purpose:\n");
  process.stdout.write("  Manage autonomous OpenCode runs with GitHub Actions workflows.\n");
  process.stdout.write("  OpenCode users only: tender reuses your existing opencode.json/.opencode setup.\n");
  process.stdout.write("  Workflow files are the source of truth (no sidecar metadata files).\n\n");
  process.stdout.write("Coding Agent Guide:\n");
  process.stdout.write("  1. Inspect current tenders with `ls`.\n");
  process.stdout.write("  2. Use `add` or `update` to define automation declaratively.\n");
  process.stdout.write("  3. Use `run` to trigger workflow_dispatch runs immediately.\n");
  process.stdout.write("  4. Commit generated workflow changes under .github/workflows.\n\n");
  process.stdout.write("Use Cases:\n");
  process.stdout.write("  - Push target test trees and run an agent that diffs target vs actual, then implements.\n");
  process.stdout.write("  - Run a test review + improvement agent hourly in a busy trunk-based codebase.\n");
  process.stdout.write("  - Run an intent-capture flow: inspect rough code, write formalizing tests, then reimplement TDD-style.\n\n");
  process.stdout.write("Advice For Coding Agents:\n");
  process.stdout.write("  After reading this guide, propose creative new tender workflows for this repo.\n");
  process.stdout.write("  Share those ideas with the user before making major automation changes.\n\n");
  process.stdout.write("Usage:\n");
  process.stdout.write("  pnpm dlx @susu-eng/tender@latest\n");
  process.stdout.write("  npx @susu-eng/tender@latest <command>\n");
  process.stdout.write("  npx @susu-eng/tender@latest help <command>\n");
  process.stdout.write("  npx @susu-eng/tender@latest <command> --help\n\n");
  process.stdout.write("Commands:\n");
  process.stdout.write("  init            Ensure .github/workflows exists\n");
  process.stdout.write("  add             Add a tender non-interactively (agent-friendly)\n");
  process.stdout.write("  update          Update a tender non-interactively (agent-friendly)\n");
  process.stdout.write("  ls              List managed tender workflows\n");
  process.stdout.write("  run             Trigger an on-demand tender now via GitHub CLI\n");
  process.stdout.write("  rm              Remove a tender workflow\n");
  process.stdout.write("  help [command]  Show command help\n\n");
  process.stdout.write("Examples:\n");
  process.stdout.write("  npx @susu-eng/tender@latest ls\n");
  process.stdout.write("  npx @susu-eng/tender@latest add --name nightly --agent Build --cron \"0 9 * * 1\"\n");
  process.stdout.write("  npx @susu-eng/tender@latest run nightly --prompt \"review and commit\"\n");
  process.stdout.write("  npx @susu-eng/tender@latest help add\n");
}

async function resolveBinaryPath() {
  if (process.env.TENDER_BINARY_PATH) {
    if (!fs.existsSync(process.env.TENDER_BINARY_PATH)) {
      throw new Error(`TENDER_BINARY_PATH not found: ${process.env.TENDER_BINARY_PATH}`);
    }
    return process.env.TENDER_BINARY_PATH;
  }

  const ext = process.platform === "win32" ? ".exe" : "";
  const binaryName = `tender${ext}`;
  const candidates = [
    path.join(process.cwd(), "bin", binaryName),
    path.join(__dirname, "..", "bin", binaryName),
  ];
  for (const candidate of candidates) {
    if (fs.existsSync(candidate)) {
      return candidate;
    }
  }

  return await ensureDownloadedBinary();
}

function execBinary(binPath, binaryArgs) {
  return new Promise((resolve, reject) => {
    const child = spawn(binPath, binaryArgs, { stdio: "inherit" });

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

async function ensureDownloadedBinary() {
  const mappedPlatform = mapPlatform(process.platform);
  const mappedArch = mapArch(process.arch);
  if (!mappedPlatform || !mappedArch) {
    throw new Error(
      `Unsupported platform for @susu-eng/tender: platform=${process.platform} arch=${process.arch}`
    );
  }

  const ext = mappedPlatform === "windows" ? ".exe" : "";
  const localName = `tender${ext}`;
  const cacheDir = path.join(defaultCacheRoot(), "tender", "cli", packageVersion);
  const cachePath = path.join(cacheDir, localName);
  if (fs.existsSync(cachePath)) {
    return cachePath;
  }

  fs.mkdirSync(cacheDir, { recursive: true });

  const assetName = `tender_${packageVersion}_${mappedPlatform}_${mappedArch}${ext}`;
  const downloadUrl = `${releaseBaseUrl}/v${packageVersion}/${assetName}`;
  const tempPath = path.join(cacheDir, `${localName}.tmp-${process.pid}`);

  try {
    await downloadToPath(downloadUrl, tempPath);
    if (mappedPlatform !== "windows") {
      fs.chmodSync(tempPath, 0o755);
    }
    fs.renameSync(tempPath, cachePath);
  } catch (error) {
    try {
      fs.rmSync(tempPath, { force: true });
    } catch {
      // Best-effort temp cleanup only.
    }
    if (fs.existsSync(cachePath)) {
      return cachePath;
    }
    throw new Error(
      [
        `Unable to download tender binary for ${mappedPlatform}/${mappedArch} (${packageVersion}).`,
        `Tried: ${downloadUrl}`,
        `Original error: ${error instanceof Error ? error.message : String(error)}`,
      ].join(" ")
    );
  }

  return cachePath;
}

function defaultCacheRoot() {
  if (process.env.TENDER_CACHE_DIR) {
    return process.env.TENDER_CACHE_DIR;
  }
  if (process.platform === "win32") {
    return process.env.LOCALAPPDATA || path.join(os.homedir(), "AppData", "Local");
  }
  if (process.platform === "darwin") {
    return path.join(os.homedir(), "Library", "Caches");
  }
  return process.env.XDG_CACHE_HOME || path.join(os.homedir(), ".cache");
}

function mapPlatform(platform) {
  if (platform === "darwin") {
    return "darwin";
  }
  if (platform === "linux") {
    return "linux";
  }
  if (platform === "win32") {
    return "windows";
  }
  return "";
}

function mapArch(arch) {
  if (arch === "x64") {
    return "amd64";
  }
  if (arch === "arm64") {
    return "arm64";
  }
  return "";
}

async function downloadToPath(url, destinationPath) {
  const response = await fetch(url, {
    headers: {
      "user-agent": "@susu-eng/tender",
      accept: "application/octet-stream",
    },
    redirect: "follow",
  });
  if (!response.ok) {
    throw new Error(`HTTP ${response.status} ${response.statusText}`);
  }
  if (!response.body) {
    throw new Error("empty response body");
  }

  await streamToFile(Readable.fromWeb(response.body), destinationPath);
}

function streamToFile(readable, destinationPath) {
  return new Promise((resolve, reject) => {
    const writable = fs.createWriteStream(destinationPath);
    readable.on("error", reject);
    writable.on("error", reject);
    writable.on("finish", resolve);
    readable.pipe(writable);
  });
}
