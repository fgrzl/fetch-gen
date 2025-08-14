import { defineConfig } from "tsup";

export default defineConfig({
  entry: ["src/shim.ts"],
  format: ["cjs", "esm"],
  outDir: "dist",
  target: "node16",
  clean: false, // Don't clean the entire dist folder (it has Go binaries)
  banner: {
    js: "#!/usr/bin/env node"
  },
  shims: true, // Enable shims for import.meta, __dirname, etc.
  outExtension({ format }) {
    return {
      js: format === "cjs" ? ".cjs" : ".mjs"
    };
  }
});
