# Contributing

Thanks for contributing to @fgrzl/fetch-gen.

## Setup

1. Fork and clone the repository.
2. `npm install`
3. `npm test` (if defined) or run the CLI against fixture OpenAPI files.

## Pull requests

- Update `docs/cli-reference.md` when adding flags or output shape changes.
- Regenerate golden fixtures when codegen output changes intentionally.
- Keep generated TypeScript compatible with the current [@fgrzl/fetch](https://github.com/fgrzl/fetch) major version.

## Changelog

Note changes under `## [Unreleased]` in [CHANGELOG.md](CHANGELOG.md).
