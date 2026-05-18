# Overview

fetch-gen reads an **OpenAPI 3** document and emits a TypeScript module that wraps `@fgrzl/fetch` with:

- One function per operation (typed request/response)
- `createAdapter(client)` factory for dependency injection
- Optional custom client instance import path

## Workflow

```text
 openapi.yaml  ──►  fetch-gen  ──►  generated.ts  ──►  createAdapter(fetchClient)
```

Generated code does not embed base URLs or auth — configure the shared `FetchClient` once in your app bootstrap.

## Design goals

- **Thin generated layer** — no runtime beyond fetch and types
- **Regeneratable** — safe to overwrite output on schema changes
- **mesh-core pattern** — `gen:auth`, `gen:apiv1` scripts use `npx @fgrzl/fetch-gen`

## Versioning

Pin fetch-gen in CI when reproducible builds matter; mesh-core currently resolves latest at codegen time via `npx`.
