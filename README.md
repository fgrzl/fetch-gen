[![ci](https://github.com/fgrzl/fetch-gen/actions/workflows/ci.yml/badge.svg)](https://github.com/fgrzl/fetch-gen/actions/workflows/ci.yml)
[![publish](https://github.com/fgrzl/fetch-gen/actions/workflows/publish.yml/badge.svg)](https://github.com/fgrzl/fetch-gen/actions/workflows/publish.yml)
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

#### create custom script in package.json

```json
{
  "scripts": {
    "fetch-gen": "npx @fgrzl/fetch-gen --input openapi.yaml --output ./src/api.ts"
  }
}
```

## Generated API Usage

The generated API returns `FetchResponse<T>` objects that include both data and metadata:

```typescript
import { createApi } from './src/api';
import api from '@fgrzl/fetch';

const apiClient = createApi(api);

const response = await apiClient.getUser();
if (response.ok) {
  console.log(response.data);     // Your typed data
  console.log(response.status);   // HTTP status code
  console.log(response.headers);  // Response headers
} else {
  console.error(response.error?.message);
}
```
