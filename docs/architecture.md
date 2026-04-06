# Architecture

## Overview

Monorepo with 10 MCP servers in Go, designed as lightweight replacements for Node.js-based MCP servers.

```
Client (Claude, Nanoclaw, etc.)
    │
    ▼
┌─────────────────────┐
│  Aggregator (:8080)  │  MCP-to-MCP proxy, tool prefixing
└────────┬────────────┘
         │ Streamable HTTP
    ┌────┼────┬────┬────┬────┬────┬────┬────┐
    ▼    ▼    ▼    ▼    ▼    ▼    ▼    ▼    ▼
  brave bunq cloud disc dock gith hetz ms365 memory
  :8000 :8000 :8000 :8000 :8000 :8000 :8000 :8000 :8080
```

## Server Types

### MCP Servers (using official SDK)
Use `internal/server/` bootstrap with `modelcontextprotocol/go-sdk`:
- **brave** — Brave Search API (Web Search + Suggest)
- **bunq** — Bunq Banking API (readonly: accounts, payments, cards, schedules)
- **discord** — Discord Bot API (guilds, channels, roles, reactions, threads)
- **docker** — Docker Engine API (full r/w via Unix socket)
- **github** — GitHub REST API (readonly: repos, issues, PRs, actions, releases, search)
- **hetzner** — Hetzner Cloud API (readonly: all resource types)
- **ms365** — Microsoft Graph API (full r/w + shared mailbox, OAuth client credentials)

### Proxies (no MCP SDK)
- **cloudflare** — HTTP reverse proxy to `mcp.cloudflare.com` with token injection
- **aggregator** — MCP-to-MCP proxy, discovers tools from backends and exposes them with prefixed names

### External (not in this repo)
- **memory** — okooo5km/memory-mcp-server-go, only a Dockerfile in `docker/memory.Dockerfile`
- **gitea** — docker.gitea.com/gitea-mcp-server (external Go binary)

## Transport

All servers use **Streamable HTTP** transport:
- MCP endpoint: `POST /mcp`
- Health check: `GET /health`
- Default port: `8000` (aggregator: `8080`)

## Authentication per Server

| Server | Env Variable | Auth Method |
|--------|-------------|-------------|
| brave | `BRAVE_API_KEY` | `X-Subscription-Token` header |
| bunq | `BUNQ_API_KEY` | `X-Bunq-Client-Authentication` header |
| cloudflare | `CLOUDFLARE_API_TOKEN` | `Authorization: Bearer` (injected by proxy) |
| discord | `DISCORD_BOT_TOKEN` | `Authorization: Bot` header |
| docker | `DOCKER_HOST` (optional) | Unix socket (no auth) |
| github | `GITHUB_PERSONAL_ACCESS_TOKEN` | `Authorization: Bearer` header |
| hetzner | `HETZNER_API_TOKEN` | `Authorization: Bearer` header |
| ms365 | `MS365_CLIENT_ID`, `MS365_CLIENT_SECRET`, `MS365_TENANT_ID` | OAuth 2.0 client credentials |
| aggregator | `MCP_BACKENDS` | N/A (connects to other servers) |

## Docker Strategy

Each server gets its own Docker image built from a shared pattern:
1. `golang:1.24-alpine` builder stage
2. `CGO_ENABLED=0` static binary
3. `scratch` final image (~8-10MB)

Exception: `memory.Dockerfile` installs an external Go binary.

## Aggregator Backend Format

```
MCP_BACKENDS=brave=http://brave:8000/mcp,github=http://github:8000/mcp,...
```

Tools are prefixed with backend name: `brave_web_search`, `github_list_repos`, etc.

## Integration with infrastructure-home

Replace the Node.js services in `docker-compose.yml` with the Go binaries. Each server drops from ~300MB (node:22-slim) to ~10MB (scratch).
