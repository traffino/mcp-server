---
date: 2026-05-12
type: concept
tags: [mcp, aws, proxy, server, architecture]
---

# aws-docs-mcp-server (Phase A-Wrapper)

HTTP-Proxy zum gehosteten AWS Knowledge MCP. Anonym, ohne Auth — analog [[drawio]]-Proxy.

## Abgrenzung zu [[aws-mcp-server]]

| | `aws-docs` (dieser Proxy) | `aws` (Phase C, Account) |
|---|---|---|
| Wissen | "wie geht AWS Service X" (Doku, Best Practices) | "was lebt in MEINEM Account" |
| Upstream | `https://knowledge-mcp.global.api.aws` (gehostet) | direkt AWS API via SDK Go v2 |
| Typ | HTTP Proxy | MCP Server (SDK-basiert) |
| Auth | keine | Static IAM Access Keys |

## Warum wrappen

Phase A war urspruenglich direkt in `~/.claude.json` registriert (User-Scope, beide Realms). Wrappen jetzt aus drei Gruenden:

1. **Aggregator-Buendelung** — alle MCPs durch lokalen `aggregator`-Container, einheitlicher Einstiegspunkt.
2. **Realm-Scoping** — User-Scope-Registrierung gilt sonst beide Realms; ueber Aggregator-Compose pro Realm steuerbar.
3. **Konsistenz** — `cloudflare` und `drawio` sind ebenfalls Proxies (Auth bzw. anonym). aws-docs reiht sich ein.

## Architektur

| | |
|---|---|
| Sprache | Go (`net/http`, stdlib only) |
| Pattern | Klon von [[drawio]]-Proxy (anonym, kein Auth-Header) |
| Codeumfang | `cmd/aws-docs/main.go` (~80 Zeilen) |
| Port | `:8000` (Streamable HTTP, `/` und `/health`) |
| Header | `Content-Type`, `Accept`, optional `Mcp-Session-Id` durchgereicht |

## Tools (durchgereicht, 6 Stueck)

| Tool | Zweck |
|---|---|
| `search_documentation` | AWS-Doku-Suche, optional Topic-Filter |
| `read_documentation` | Doku-Seite zu Markdown |
| `recommend` | Content-Empfehlungen |
| `list_regions` | AWS-Regionen |
| `get_regional_availability` | Service/API/CFN-Resource-Verfuegbarkeit pro Region |
| `retrieve_skill` | Domain-spezifische Agent-Skills (z.B. Strands Agents) |

Detail: `docs/api-coverage/aws-docs.md`.

## Konfiguration

| Env-Var | Default | Zweck |
|---|---|---|
| `PORT` | `:8000` | Listen-Adresse |

Kein API-Token, kein AWS-Account, keine IAM. Rein gefaehrlich-arm.

## Build-Status (V1)

- `make aws-docs` baut sauber, Binary ~6 MB.
- `make docker-aws-docs` baut Image ~23 MB (Alpine 3.21 + tzdata).
- Smoke: `/health` antwortet `{"status":"ok","target":"https://knowledge-mcp.global.api.aws"}`. `initialize`-RPC liefert Upstream-Response `AWSKnowledgeMCP 1.0.0` (protocolVersion 2025-03-26).

## Out-of-Scope dieser Iteration

- Compose-Eintrag in `infrastructure-home` — User integriert manuell (andere Session).
- Umkonfiguration von `~/.claude.json` (Remote-Endpoint raus, Aggregator rein) — separater Schritt nach Compose.
