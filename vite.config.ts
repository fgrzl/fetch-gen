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
