#!/usr/bin/env node

import { spawnSync } from 'child_process';
import { existsSync } from 'fs';
import { arch, platform } from 'os';
import { join, dirname } from 'path';

// Works in both ESM and CJS
let __dirname: string;
try {
  // ESM context
  const { fileURLToPath } = await import('url');
  __dirname = dirname(fileURLToPath(import.meta.url));
} catch {
  // CJS context (or fallback)
  __dirname = typeof __dirname !== 'undefined' ? __dirname : process.cwd();
}

const platformMap: Record<string, string> = {
  win32: 'win32',
  darwin: 'darwin',
  linux: 'linux',
};

const archMap: Record<string, string> = {
  x64: 'x64',
  arm64: 'arm64',
};

const plat = platform();
const architecture = arch();

const platformKey = `${platformMap[plat]}-${archMap[architecture]}`;
const binaryName = plat === 'win32' ? 'fetch-gen.exe' : 'fetch-gen';
const binaryPath = join(__dirname, '..', 'dist', platformKey, binaryName);

if (!existsSync(binaryPath)) {
  console.error(`❌ fetch-gen: No binary found for platform ${platformKey}`);
  process.exit(1);
}

const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
});

process.exit(result.status ?? 1);
