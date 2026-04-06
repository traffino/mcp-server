# mcp-server

Lightweight MCP (Model Context Protocol) servers written in Go. Designed as resource-efficient replacements for Node.js-based MCP servers.

## Motivation

Most MCP servers run on Node.js and consume 200-500MB RAM each. With 10+ servers, that's 5GB+ just for MCP. These Go implementations use 10-30MB each in scratch Docker images.

## Servers

| Server | API | Scope | Status |
|--------|-----|-------|--------|
| brave | Brave Search | Web Search + Suggest | done |
| memory | Knowledge Graph | Full (external binary) | done |
| hetzner | Hetzner Cloud | All resources, read-only | done |
| cloudflare | Cloudflare | Proxy to mcp.cloudflare.com | done |
| github | GitHub REST | Repos, Issues, PRs, Actions, Releases, Search | done |
| discord | Discord Bot | Guilds, Channels, Roles, Reactions, Threads | done |
| docker | Docker Engine | Full read/write | done |
| ms365 | Microsoft Graph | Full + Shared Mailbox | done |
| bunq | Bunq Banking | Accounts, Payments, Cards, Schedules (read-only) | done |
| aggregator | MCP Proxy | Tool aggregation across backends | done |

## Quick Start

```bash
# Build all
make

# Build specific server
make brave

# Build Docker image
make docker-brave

# Run
BRAVE_API_KEY=xxx ./build/brave
```

## Docker

Each server produces a minimal scratch-based Docker image (~10-15MB):

```bash
make docker-brave
docker run -e BRAVE_API_KEY=xxx -p 8000:8000 traffino/mcp-brave
```

## Architecture

- **SDK**: Official Go MCP SDK (`modelcontextprotocol/go-sdk`, Tier 1)
- **Transport**: Streamable HTTP on `/mcp`
- **Health**: `GET /health` on every server
- **Config**: Environment variables only
- **Base image**: `scratch` (static Go binary + CA certificates)

## API Coverage

See `docs/api-coverage/` for detailed endpoint coverage per server.
