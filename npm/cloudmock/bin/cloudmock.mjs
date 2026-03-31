#!/usr/bin/env node

import { spawn } from "node:child_process";
import { chmod, mkdir, stat, writeFile } from "node:fs/promises";
import { createWriteStream } from "node:fs";
import { homedir, platform, arch } from "node:os";
import { join } from "node:path";
import { pipeline } from "node:stream/promises";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { version } = require("../package.json");

const REPO = "neureaux/cloudmock";

// Map Node.js platform/arch to release binary names.
const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const ARCH_MAP = {
  arm64: "arm64",
  x64: "amd64",
};

function getBinaryName() {
  const os = PLATFORM_MAP[platform()];
  const cpu = ARCH_MAP[arch()];

  if (!os || !cpu) {
    console.error(
      `Unsupported platform: ${platform()} ${arch()}. ` +
        `Supported: darwin/linux/win32 on arm64/x64.`
    );
    process.exit(1);
  }

  const ext = os === "windows" ? ".exe" : "";
  return `cloudmock-${os}-${cpu}${ext}`;
}

function getCachePath(binaryName) {
  return join(homedir(), ".cloudmock", "bin", `${binaryName}-${version}`);
}

async function exists(path) {
  try {
    await stat(path);
    return true;
  } catch {
    return false;
  }
}

async function download(binaryName, dest) {
  const url = `https://github.com/${REPO}/releases/download/v${version}/${binaryName}`;

  console.log(`CloudMock v${version} not cached. Downloading...`);
  console.log(`  ${url}`);

  // Ensure cache directory exists.
  await mkdir(join(homedir(), ".cloudmock", "bin"), { recursive: true });

  let response;
  try {
    response = await fetch(url, { redirect: "follow" });
  } catch (err) {
    console.error(`\nDownload failed: ${err.message}\n`);
    console.error("Alternatives:");
    console.error("  docker run -p 4566:4566 ghcr.io/neureaux/cloudmock");
    console.error(
      "  go install github.com/neureaux/cloudmock/cmd/gateway@latest"
    );
    process.exit(1);
  }

  if (!response.ok) {
    console.error(`\nDownload failed: HTTP ${response.status}`);
    console.error(`  URL: ${url}\n`);
    console.error("Alternatives:");
    console.error("  docker run -p 4566:4566 ghcr.io/neureaux/cloudmock");
    console.error(
      "  go install github.com/neureaux/cloudmock/cmd/gateway@latest"
    );
    process.exit(1);
  }

  const fileStream = createWriteStream(dest);
  await pipeline(response.body, fileStream);

  // Make executable (no-op on Windows).
  if (platform() !== "win32") {
    await chmod(dest, 0o755);
  }

  console.log("  Done.\n");
}

async function main() {
  const binaryName = getBinaryName();
  const cached = getCachePath(binaryName);

  if (!(await exists(cached))) {
    await download(binaryName, cached);
  }

  const child = spawn(cached, process.argv.slice(2), {
    stdio: "inherit",
  });

  child.on("error", (err) => {
    console.error(`Failed to start CloudMock: ${err.message}`);
    process.exit(1);
  });

  child.on("exit", (code, signal) => {
    if (signal) {
      process.kill(process.pid, signal);
    } else {
      process.exit(code ?? 1);
    }
  });
}

main();
