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

Generate your API client to `src/adapters/generated.ts`:

```bash
npx @fgrzl/fetch-gen --input openapi.yaml --output ./src/adapters/generated.ts
```

### Simple Setup

Create `src/adapters/index.ts` for basic usage:

```typescript
import { createAdapter } from './generated';
import client from '@fgrzl/fetch';

// Set the base URL and create your adapter
client.setBaseUrl('https://api.example.com');
const myAdapter = createAdapter(client);

export default myAdapter;
```

Now you can use it throughout your application:

```typescript
import myAdapter from './src/adapters';

const response = await myAdapter.getUser();
if (response.ok) {
  console.log(response.data); // Your typed data
} else {
  console.error(response.error?.message);
}
```

### Advanced Configuration

For production applications, you can add authentication, retry logic, and other middleware:

```typescript
import { createAdapter } from './generated';
import { FetchClient, useAuthentication, useRetry } from '@fgrzl/fetch';

// Create base client with full configuration
let client = new FetchClient({
  baseUrl: 'https://api.example.com',
  credentials: 'same-origin',
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add authentication
client = useAuthentication(client, {
  tokenProvider: () => localStorage.getItem('auth-token') || '',
});

// Add retry logic
client = useRetry(client, {
  maxRetries: 3,
  delay: 1000,
  backoff: 'exponential',
});

// Create the adapter
const myAdapter = createAdapter(client);

export default myAdapter;
```
