#!/usr/bin/env node

const { spawnSync } = require('child_process');
const { existsSync } = require('fs');
const { arch, platform } = require('os');
const { join, dirname } = require('path');

// Fallback for __dirname in ESM-like contexts (e.g., npx)
function resolveDirname() {
  try {
    // If running in ESM (import.meta.url available)
    const { fileURLToPath } = require('url');
    return dirname(fileURLToPath(import.meta.url));
  } catch {
    // If __dirname is defined (CommonJS), use it
    if (typeof __dirname !== 'undefined') return __dirname;
    // Last resort
    return process.cwd();
  }
}

const __dirnameResolved = resolveDirname();

const platformMap = {
  win32: 'win32',
  darwin: 'darwin',
  linux: 'linux',
};

const archMap = {
  x64: 'x64',
  arm64: 'arm64',
};

const plat = platform();
const architecture = arch();

const platformKey = `${platformMap[plat]}-${archMap[architecture]}`;
const binaryName = plat === 'win32' ? 'fetch-gen.exe' : 'fetch-gen';
const binaryPath = join(
  __dirnameResolved,
  '..',
  'dist',
  platformKey,
  binaryName
);

if (!existsSync(binaryPath)) {
  console.error(`❌ fetch-gen: No binary found for platform ${platformKey}`);
  process.exit(1);
}

const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
});

process.exit(result.status ?? 1);
