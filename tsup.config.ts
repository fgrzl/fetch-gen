import { defineConfig } from 'tsup';

export default defineConfig({
  entry: ['src/shim.ts'],
  format: ['cjs', 'esm'],
  outDir: 'dist',
  target: 'node16',
  clean: false, // Preserve Go binaries in dist/ by not cleaning the entire directory
  banner: {
    js: '#!/usr/bin/env node',
  },
  shims: true, // Enable shims for import.meta, __dirname, etc.
  outExtension({ format }) {
    return {
      js: format === 'cjs' ? '.cjs' : '.mjs',
    };
  },
});
