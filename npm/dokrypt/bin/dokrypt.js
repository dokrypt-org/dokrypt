#!/usr/bin/env node

"use strict";

const { execFileSync } = require("child_process");
const path = require("path");

const PLATFORMS = {
  "win32 x64": "@dokrypt/win32-x64/bin/dokrypt.exe",
  "darwin arm64": "@dokrypt/darwin-arm64/bin/dokrypt",
  "darwin x64": "@dokrypt/darwin-x64/bin/dokrypt",
  "linux x64": "@dokrypt/linux-x64/bin/dokrypt",
  "linux arm64": "@dokrypt/linux-arm64/bin/dokrypt",
};

const key = `${process.platform} ${process.arch}`;
const binPath = PLATFORMS[key];

if (!binPath) {
  console.error(
    `dokrypt: unsupported platform ${process.platform}/${process.arch}\n` +
      `Supported: windows/x64, macOS/arm64, macOS/x64, linux/x64, linux/arm64\n` +
      `Install from source: go install github.com/dokrypt-org/dokrypt/cmd/dokrypt@latest`
  );
  process.exit(1);
}

let binary;
try {
  binary = require.resolve(binPath);
} catch {
  console.error(
    `dokrypt: could not find binary package for ${process.platform}/${process.arch}\n` +
      `Expected package: ${binPath.split("/bin/")[0]}\n\n` +
      `Try reinstalling: npm install -g dokrypt`
  );
  process.exit(1);
}

try {
  execFileSync(binary, process.argv.slice(2), { stdio: "inherit" });
} catch (e) {
  if (e.status !== undefined) {
    process.exit(e.status);
  }
  throw e;
}
