# CLI reference

```bash
npx @fgrzl/fetch-gen --input <openapi.yaml> --output <path.ts> [options]
```

## Required flags

| Flag | Description |
|------|-------------|
| `--input` | Path to OpenAPI 3 YAML or JSON |
| `--output` | TypeScript file to write |

## Optional flags

| Flag | Description |
|------|-------------|
| `--instance` | Import path to a custom fetch client module (default: `@fgrzl/fetch`) |

## Output

- TypeScript types for schemas referenced by operations
- Functions named from `operationId` (or derived names)
- `createAdapter(client)` export

## Regeneration

Overwrite the output file on each run. Do not hand-edit generated files — adjust the OpenAPI spec or generator version instead.

## Troubleshooting

- Ensure `operationId` is set for stable function names
- Resolve schema `$ref` issues in the OpenAPI document before codegen
- Match `@fgrzl/fetch` major version with what fetch-gen expects (see fetch release notes)
