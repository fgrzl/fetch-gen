# Getting started

## Generate a client

```bash
npx @fgrzl/fetch-gen --input openapi.yaml --output ./src/adapters/generated.ts
```

## package.json script

```json
{
  "scripts": {
    "gen:api": "npx @fgrzl/fetch-gen --input ./openapi.yaml --output ./src/adapters/generated.ts"
  }
}
```

## Use the adapter

```typescript
import { createAdapter } from './adapters/generated';
import client from '@fgrzl/fetch';

client.setBaseUrl('https://api.example.com');
const api = createAdapter(client);

const res = await api.getUser({ id: '1' });
if (res.ok) console.log(res.data);
```

## Custom client module

```bash
npx @fgrzl/fetch-gen --input openapi.yaml --output ./src/api.ts --instance ./src/custom
```

Pass a module that exports your preconfigured `FetchClient` (middleware, auth, retry).

## Related

- [CLI reference](cli-reference.md)
- [fetch getting started](https://github.com/fgrzl/fetch/blob/main/docs/getting-started.md)
