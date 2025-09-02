import { describe, it, expect, beforeAll, afterEach, afterAll } from 'vitest';
import { spawn, spawnSync } from 'child_process';
import { existsSync, rmSync, mkdirSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const projectRoot = join(__dirname, '..');
const testOutput = join(__dirname, 'temp');

// Dynamically determine the Go binary path based on platform and arch
// This should match exactly how the shim itself determines the path
function getGoBinaryPath() {
  const platformMap: Record<string, string> = {
    win32: 'win32',
    darwin: 'darwin',
    linux: 'linux',
  };

  const archMap: Record<string, string> = {
    x64: 'x64',
    arm64: 'arm64',
  };

  const plat = process.platform;
  const architecture = process.arch;

  const platformKey = `${platformMap[plat] || plat}-${archMap[architecture] || architecture}`;
  const binaryName = plat === 'win32' ? 'fetch-gen.exe' : 'fetch-gen';

  return join(projectRoot, 'dist', platformKey, binaryName);
}

describe('Shim Integration Tests', () => {
  beforeAll(async () => {
    // Check if shims exist, build them if not
    const cjsExists = existsSync(join(projectRoot, 'dist/shim.cjs'));
    const mjsExists = existsSync(join(projectRoot, 'dist/shim.mjs'));

    if (!cjsExists || !mjsExists) {
      const buildResult = spawnSync('npm', ['run', 'build:shims'], {
        cwd: projectRoot,
        stdio: 'pipe',
      });

      if (buildResult.status !== 0) {
        const errorMsg =
          buildResult.stderr?.toString() ||
          buildResult.stdout?.toString() ||
          'Unknown error';
        throw new Error(`Failed to build shims: ${errorMsg}`);
      }
    }

    // Ensure Go binary exists
    const goBinaryPath = getGoBinaryPath();
    if (!existsSync(goBinaryPath)) {
      const goBuildResult = spawnSync(
        'go',
        ['build', '-o', goBinaryPath, './cmd'],
        {
          cwd: projectRoot,
          stdio: 'pipe',
          shell: true,
        }
      );

      if (goBuildResult.status !== 0) {
        console.warn('Go binary build failed - some tests may be skipped');
      }
    }

    // Create clean temp directory for test outputs
    if (existsSync(testOutput)) {
      try {
        rmSync(testOutput, { recursive: true, force: true });
      } catch (error) {
        console.warn(`Failed to clean existing test output: ${error}`);
      }
    }
    mkdirSync(testOutput, { recursive: true });
  });

  afterEach(() => {
    // Clean up test output files after each test
    if (existsSync(testOutput)) {
      try {
        rmSync(testOutput, { recursive: true, force: true });
      } catch (error) {
        console.warn(`Failed to clean up test output: ${error}`);
      }
    }
  });

  afterAll(() => {
    // Final cleanup - remove test directory completely
    if (existsSync(testOutput)) {
      try {
        rmSync(testOutput, { recursive: true, force: true });
      } catch (error) {
        console.warn(`Failed to clean up test directory: ${error}`);
      }
    }
  });

  describe('Shim Files Existence', () => {
    it('should have generated shim.cjs', () => {
      const cjsPath = join(projectRoot, 'dist/shim.cjs');
      expect(existsSync(cjsPath)).toBe(true);
    });

    it('should have generated shim.mjs', () => {
      const mjsPath = join(projectRoot, 'dist/shim.mjs');
      expect(existsSync(mjsPath)).toBe(true);
    });

    it('shim.cjs should have executable shebang', async () => {
      const { readFile } = await import('fs/promises');
      const cjsPath = join(projectRoot, 'dist/shim.cjs');
      const content = await readFile(cjsPath, 'utf-8');
      expect(content.startsWith('#!/usr/bin/env node')).toBe(true);
    });

    it('shim.mjs should have executable shebang', async () => {
      const { readFile } = await import('fs/promises');
      const mjsPath = join(projectRoot, 'dist/shim.mjs');
      const content = await readFile(mjsPath, 'utf-8');
      expect(content.startsWith('#!/usr/bin/env node')).toBe(true);
    });
  });

  describe('Platform Detection', () => {
    it('should handle platform detection correctly', () => {
      const cjsPath = join(projectRoot, 'dist/shim.cjs');

      // Run shim with --help to test basic functionality
      const result = spawnSync('node', [cjsPath, '--help'], {
        stdio: 'pipe',
      });

      // Should exit with non-zero (help/usage), but not crash
      expect(result.status).not.toBe(0);
      const output =
        result.stderr?.toString() || result.stdout?.toString() || '';

      // Should either show usage (if binary found) or platform error (if not found)
      const hasUsage = output.includes('Usage:');
      const hasPlatformError = output.includes('No binary found for platform');
      const hasErrorMessage = output.includes('Error:');

      expect(hasUsage || hasPlatformError || hasErrorMessage).toBe(true);
    });
  });

  describe('Real Binary Integration', () => {
    const goBinaryPath = getGoBinaryPath();

    it.skipIf(!existsSync(goBinaryPath))(
      'should execute real Go binary through CJS shim',
      () => {
        const cjsPath = join(projectRoot, 'dist/shim.cjs');
        const inputPath = join(projectRoot, 'openapi-test.yaml');
        const outputPath = join(testOutput, 'test-output.ts');

        const result = spawnSync(
          'node',
          [cjsPath, '--input', inputPath, '--output', outputPath],
          {
            stdio: 'pipe',
          }
        );

        expect(result.status).toBe(0);
        expect(existsSync(outputPath)).toBe(true);
      }
    );

    it.skipIf(!existsSync(goBinaryPath))(
      'should execute real Go binary through ESM shim',
      () => {
        const mjsPath = join(projectRoot, 'dist/shim.mjs');
        const inputPath = join(projectRoot, 'openapi-test.yaml');
        const outputPath = join(testOutput, 'test-output-esm.ts');

        const result = spawnSync(
          'node',
          [mjsPath, '--input', inputPath, '--output', outputPath],
          {
            stdio: 'pipe',
          }
        );

        expect(result.status).toBe(0);
        expect(existsSync(outputPath)).toBe(true);
      }
    );

    it.skipIf(!existsSync(goBinaryPath))(
      'should pass arguments correctly to Go binary',
      async () => {
        const cjsPath = join(projectRoot, 'dist/shim.cjs');
        const inputPath = join(projectRoot, 'openapi-test.yaml');
        const outputPath = join(testOutput, 'test-args.ts');

        const result = spawnSync(
          'node',
          [cjsPath, '--input', inputPath, '--output', outputPath],
          {
            stdio: 'pipe',
          }
        );

        expect(result.status).toBe(0);

        // Verify the generated file has expected content
        if (existsSync(outputPath)) {
          const { readFileSync } = await import('fs');
          const content = readFileSync(outputPath, 'utf-8');
          expect(content).toContain('export function createAdapter');
          expect(content).toContain('getUsers');
          expect(content).toContain('getUserById');
        }
      }
    );
  });

  describe('Error Handling', () => {
    it('should exit with error code when Go binary fails', () => {
      const cjsPath = join(projectRoot, 'dist/shim.cjs');

      // Pass invalid arguments to cause Go binary to fail
      const result = spawnSync(
        'node',
        [
          cjsPath,
          '--input',
          'nonexistent-file.yaml',
          '--output',
          '/invalid/path/output.ts',
        ],
        {
          stdio: 'pipe',
        }
      );

      expect(result.status).not.toBe(0);
    });

    it('should show appropriate message when binary not found', () => {
      const cjsPath = join(projectRoot, 'dist/shim.cjs');

      const result = spawnSync('node', [cjsPath, 'test'], {
        stdio: 'pipe',
      });

      expect(result.status).toBe(1);
      const output =
        result.stderr?.toString() || result.stdout?.toString() || '';

      // Should either show usage (if binary found) or platform error (if not found)
      const hasUsage = output.includes('Usage:');
      const hasPlatformError = output.includes('No binary found for platform');
      const hasErrorMessage = output.includes('Error:');

      expect(hasUsage || hasPlatformError || hasErrorMessage).toBe(true);
    });
  });

  describe('Module Format Compatibility', () => {
    it('CJS shim should work with require()', async () => {
      // This test runs in a separate process to test actual require() usage
      const testScript = `
        const { spawnSync } = require('child_process');
        const path = require('path');
        const shimPath = path.join('${projectRoot}', 'dist', 'shim.cjs');
        const result = spawnSync('node', [shimPath, '--help'], { stdio: 'pipe' });
        process.exit(result.status === 0 ? 0 : 1);
      `;

      const result = spawnSync('node', ['-e', testScript], {
        stdio: 'pipe',
      });

      // Should fail gracefully (binary not found) rather than with module errors
      expect(result.stderr?.toString() || '').not.toContain('ERR_REQUIRE_ESM');
      expect(result.stderr?.toString() || '').not.toContain(
        'Cannot use import statement'
      );
    });

    it('ESM shim should work with dynamic import()', async () => {
      const mjsPath = join(projectRoot, 'dist/shim.mjs');

      // Test that the ESM shim can be imported (though not executed without binary)
      const testScript = `
        import('${mjsPath}').then(() => {
          console.log('ESM import successful');
          process.exit(0);
        }).catch((err) => {
          console.error('ESM import failed:', err.message);
          process.exit(1);
        });
      `;

      const result = spawnSync(
        'node',
        ['--input-type=module', '-e', testScript],
        {
          stdio: 'pipe',
        }
      );

      // The import should succeed even if execution fails due to missing binary
      expect(result.stderr?.toString() || '').not.toContain('SyntaxError');
      expect(result.stderr?.toString() || '').not.toContain(
        'ERR_MODULE_NOT_FOUND'
      );
    });
  });
});
