# mcp-server

Go MCP Server Monorepo. Ressourcenarme Alternativen zu Node.js-basierten MCP-Servern.

## Sprache

Kommuniziere auf Deutsch. Code, Commits und technische Bezeichnungen auf Englisch.

## Projekt-Struktur

```
cmd/<name>/main.go     — Ein Binary pro MCP-Server
internal/server/       — Shared MCP Bootstrap (HTTP transport, health, shutdown)
internal/config/       — Env-basierte Konfiguration
internal/httputil/     — Shared HTTP Client Helpers
docker/<name>.Dockerfile — Ein Image pro Server
docs/api-coverage/     — API-Endpoint-Status pro Server
```

## SDK

Offizielles MCP SDK: `github.com/modelcontextprotocol/go-sdk` (Tier 1).

## Build

```bash
make                   # Alle Binaries bauen
make brave             # Einzelnes Binary
make test              # Tests
make docker-brave      # Docker Image
make docker-all        # Alle Docker Images
make clean             # Build-Artefakte loeschen
```

## Konventionen

- Jeder MCP-Server nutzt `internal/server.New()` fuer Bootstrap
- Port 8000 fuer MCP (Streamable HTTP auf `/mcp`), Health auf `/health`
- Konfiguration ausschliesslich ueber Environment-Variablen
- Docker: Multi-stage Build, `alpine:3.21` als finale Base, CGO_ENABLED=0
- Keine externen Dependencies ausser dem MCP SDK und stdlib
- Sonderfaelle: cloudflare (Proxy), aggregator (Proxy), memory (externes Binary)

## Server-Uebersicht

| Server | API | Scope | Typ |
|--------|-----|-------|-----|
| brave | Brave Search | Web Search + Suggest (Free) | MCP Server |
| memory | Knowledge Graph | Vollstaendig (extern) | Nur Dockerfile |
| hetzner | Hetzner Cloud | Alle Bereiche, readonly | MCP Server |
| cloudflare | Cloudflare | Proxy zu mcp.cloudflare.com | HTTP Proxy |
| drawio | draw.io | Proxy zu mcp.draw.io (Diagramme, Shapes) | HTTP Proxy |
| github | GitHub REST | Repos, Issues, PRs, Actions, Releases, Search, Users | MCP Server |
| discord | Discord Bot | Guilds, Channels, Messages, Roles, Reactions, Threads (r/w) | MCP Server |
| docker | Docker Engine | Vollstaendig (r/w) | MCP Server |
| ms365 | Microsoft Graph | Vollstaendig + Shared Mailbox (r/w) | MCP Server |
| bunq | Bunq Banking | Konten, Payments, Cards, Schedules (readonly) | MCP Server |
| personal | Personal Productivity | Ueberstunden, Urlaub, Kranktage, People, Events, Projekte, TODOs (SQLite) | MCP Server |
| aggregator | MCP Proxy | Tool-Aggregation ueber alle Backends | HTTP Proxy |

## Deployment

Siehe `docs/architecture.md` fuer Docker Compose Integration in infrastructure-home.
