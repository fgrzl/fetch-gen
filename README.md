[![ci](https://github.com/fgrzl/fetch-gen/actions/workflows/ci.yml/badge.svg)](https://github.com/fgrzl/fetch-gen/actions/workflows/ci.yml)
[![Dependabot Updates](https://github.com/fgrzl/fetch-gen/actions/workflows/dependabot/dependabot-updates/badge.svg)](https://github.com/fgrzl/fetch-gen/actions/workflows/dependabot/dependabot-updates)
# @fgrzl/fetch-gen

Generate @fgrzl/fetch clients from OpenAPI spec. 

see -> https://github.com/fgrzl/fetch

## Usage

#### use default fetch client

```bash
npx @fgrzl/fetch-gen --input openapi.yaml --output ./src/api.ts
```

#### use a custom client defined in ./src/custom.ts

```bash
npx @fgrzl/fetch-gen --input openapi.yaml --output ./src/api.ts --instance ./src/custom
```

#### setup in you package scripts

```json
{
  "scripts": {
    "fetch-gen": "npx @fgrzl/fetch-gen --input openapi.yaml --output ./src/api.ts",
  }
}
```
