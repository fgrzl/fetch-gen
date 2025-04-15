const { platform, arch } = require('os');
const fs = require('fs');
const path = require('path');

const key = `${platform()}-${arch()}`;
const binaryName = platform() === 'win32' ? 'fetch-gen.exe' : 'fetch-gen';

const source = path.resolve(__dirname, '..', 'dist', key, binaryName);
const dest = path.resolve(__dirname, '..', 'fetch-gen');

if (!fs.existsSync(source)) {
  console.error(`❌ Binary not found for platform: ${key}`);
  console.error(`Expected binary at: ${source}`);
  process.exit(1);
}

try {
  fs.copyFileSync(source, dest);
  fs.chmodSync(dest, 0o755);
  console.log(`✅ Installed ${binaryName} to project root`);
} catch (err) {
  console.error(`❌ Failed to install binary:`, err.message);
  process.exit(1);
}
