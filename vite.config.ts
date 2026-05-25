import { defineConfig } from 'vite-plus';

export default defineConfig({
  fmt: {
    ignorePatterns: ['dist/**', 'tests/artifacts/**'],
    printWidth: 80,
    tabWidth: 2,
    semi: true,
    singleQuote: true,
    trailingComma: 'es5',
  },
  lint: {
    ignorePatterns: ['dist/**', 'tests/artifacts/**'],
    options: {
      typeAware: true,
      typeCheck: true,
    },
  },
  test: {
    globals: true,
    hookTimeout: 30000,
    environment: 'node',
    include: ['tests/**/*.{test,spec}.{js,mjs,cjs,ts,mts,cts}'],
    exclude: ['tests/fixtures/**'],
  },
  pack: {
    entry: ['src/shim.ts'],
    format: ['esm', 'cjs'],
    outDir: 'dist',
    target: 'node16',
    platform: 'node',
    clean: false,
    banner: '#!/usr/bin/env node',
    shims: true,
    fixedExtension: true,
  },
});
