# Contributing to backctl

## Project structure

```console
cmd/backctl/          CLI entry point
cmd/backctl-mcp/      MCP server entry point
internal/backstage/   Backstage API service layer
internal/client/      HTTP client (auth, retry, ETag cache)
internal/config/      Config file loading (XDG-aware)
internal/entityref/   Entity ref parser
internal/mcp/         MCP server tools and dispatch
internal/output/      Output formatting (table, JSON, YAML, tree, HTML→text)
internal/resolver/    Relationship graph traversal
internal/testutil/    Test HTTP server helper
Dockerfile            Multi-stage build (for local/manual use)
Dockerfile.goreleaser Runtime-only image (used by GoReleaser)
.goreleaser.yaml      GoReleaser configuration
```

## Prerequisites

- Go 1.25+
- Docker (for image builds)
- [GoReleaser](https://goreleaser.com/install/) (optional, for testing releases locally)

## Building

```sh
# CLI
make build

# MCP server
make build-mcp

# Docker image
make docker-build
```

## Running tests

```sh
make test
```

This runs `go test -race ./...` across all packages.

For coverage:

```sh
make coverage        # prints coverage by function
make coverage-html   # opens HTML report in browser
```

## Testing the CLI locally

```sh
export BACKSTAGE_URL=https://backstage.example.com
export BACKSTAGE_TOKEN=your-token

./bin/backctl catalog list --kind Component --limit 5
./bin/backctl catalog get component:default/some-service
./bin/backctl search "payment"
./bin/backctl relations tree component:default/some-service --depth 2
./bin/backctl techdocs list-pages component:default/some-service
```

## Testing the MCP server locally

### Direct stdin/stdout

The MCP server speaks JSON-RPC over stdio. You can validate it without an IDE:

```sh
export BACKSTAGE_URL=https://backstage.example.com
export BACKSTAGE_TOKEN=your-token

./bin/backctl-mcp <<'EOF'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search","arguments":{"query":"payment"}}}
EOF
```

Expected output: JSON-RPC responses with the tool list and search results.

### MCP Inspector

The [MCP Inspector](https://github.com/modelcontextprotocol/inspector) provides an interactive web UI to explore and invoke tools:

```sh
npx @modelcontextprotocol/inspector ./bin/backctl-mcp
```

This is the most convenient way to validate tool schemas and responses without an IDE.

### Docker

Validate the containerized server works end-to-end:

```sh
make docker-build

echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | \
  docker run --rm -i \
    -e BACKSTAGE_URL=https://backstage.example.com \
    -e BACKSTAGE_TOKEN=your-token \
    backctl-mcp:latest
```

### From Cursor

Configure `.cursor/mcp.json` as described in the README and ask the agent something like:

```text
"Find services related to payments"
```

The agent should invoke the `search` tool and display results from the catalog.

## CI/CD

The project uses GitHub Actions for continuous integration and GoReleaser for releases.

**CI** (`.github/workflows/ci.yml`) runs on every push to `main` and on pull requests:

- `go build ./...`
- `go test -race ./...`
- `go vet ./...`

**Release** (`.github/workflows/release.yml`) runs when a version tag is pushed.

## Creating a release

```sh
git tag v0.2.0
git push origin v0.2.0
```

This triggers GoReleaser, which:

1. Compiles `backctl` and `backctl-mcp` for linux/darwin (amd64 + arm64)
2. Publishes binaries and checksums to the GitHub Release
3. Builds and pushes multi-arch Docker images to `ghcr.io/jsolana/backctl-mcp:<version>` and `:latest`
4. Generates a changelog from commit messages

### Testing a release locally

```sh
goreleaser release --snapshot --clean
```

This produces all artifacts in `dist/` without publishing anything.

## Linting

```sh
make lint
```

Requires [golangci-lint](https://golangci-lint.run/welcome/install/).
