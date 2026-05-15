---
date: 2026-05-15
type: concept
tags: [mcp, forge, github, gitea, bitbucket, server, architecture]
---

# Forge-MCP-Wrapper

Drei Forge-MCP-Server (GitHub, Gitea, Bitbucket) im Monorepo, alle als Wrapper um **offizielle** Upstream-MCPs. Read + Write fuer Issues und PRs (inkl. PR-Review/Merge). Loest den vorherigen Eigenbau-GitHub-Server (Read-only, 20 Tools) ab.

## Pattern-Differenzierung

| Server | Pattern | Begruendung |
|---|---|---|
| `github` | **HTTP-Proxy** (wie [[refs/user/concepts/aws-docs-mcp-server\|aws-docs]] / cloudflare) | Upstream `api.githubcopilot.com/mcp/` ist gehostet — wir leiten Streamable-HTTP weiter, setzen Bearer-Token aus Env. |
| `bitbucket` | **HTTP-Proxy** (wie cloudflare) | Upstream `mcp.atlassian.com/v1/mcp` (Atlassian Rovo) ist gehostet. |
| `gitea` | **Externes Binary** (wie `memory`) | Gitea hostet KEINEN zentralen Remote-MCP. Offizielles `gitea-mcp`-Binary (gitea.com/gitea/gitea-mcp) spricht Streamable-HTTP von sich aus — wir bauen es aus Source und starten `-t http`. Kein eigener Go-Code, kein `cmd/gitea/`-Verzeichnis. |

`scripts/verify-mcp-matrix.sh` erlaubt explizit "pure-proxy/external servers may have only a Dockerfile" — daher kein cmd/-Marker fuer gitea.

## Doktrin-Wechsel (vs. AWS-Phase-B)

[[aws-mcp-server]] dokumentiert die alte Linie: "Phase B (verworfen) — `awslabs/aws-api-mcp-server` ... Nicht gewaehlt: gegen Lightweight-Go-Linie des Monorepos". Diese Praezedenz wurde **bewusst aufgegeben**:

- AWS hatte einen Python-Server gemeint, das ist nicht hosted und kostet Container-Pflege ohne Pflege-Ersparnis.
- GitHub/Gitea/Bitbucket haben **gehostete Remote-MCPs** (GitHub/Bitbucket) bzw. **offizielle Binaries vom Forge-Anbieter selbst** (Gitea). Selbst-Implementierung der Forge-REST-APIs reproduziert Pflege ohne Gegenwert.
- Praezedenz fuers Outsourcen an offizielle hosted MCPs existiert schon: `aws-docs` (`knowledge-mcp.global.api.aws`) wird identisch geproxied.

Die Doktrin lautet jetzt: **Eigenbau-Server wenn Forge-API self-pflegbar und kein offizieller MCP existiert. Sonst Wrapper.**

## Migration GitHub-Server (2026-05-15)

Frueher `cmd/github/main.go` 13K-LOC Read-only-REST-Wrapper, angelegt 2026-04-06 im Initial-Commit `9e5e041`. Coverage in `docs/api-coverage/github.md` (alt) markierte alle Write-Endpoints "out-of-scope".

Ersetzt durch 78-Zeilen-Proxy. Beweggrund:

- User-Scope erweitert auf Issue+PR-Write (inkl. Review/Merge) — Eigenbau-Erweiterung ~8-10 neue Tools + Token-Handling.
- Offizieller `github/github-mcp-server` (GA Sept 2025) deckt Read+Write 1:1 ab.
- Token-Scope filtert ungewollte Tools serverseitig (kein Code-Aufwand).

## Auth

| Server | Env-Var | Token-Typ | Pflichtige Scopes |
|---|---|---|---|
| `github` | `GITHUB_TOKEN` | Fine-grained PAT empfohlen | Issues r/w, Pull Requests r/w, Contents r, Metadata r |
| `gitea` | `GITEA_ACCESS_TOKEN` + `GITEA_HOST` | Gitea-PAT | read/write repository + issue, plus release fuer Releases |
| `bitbucket` | `ATLASSIAN_API_TOKEN` | Scoped API Token (KEIN App Password — wird 2026-06 abgekuendigt) | Repository r/w, PR r/w |

OAuth-Flows der Upstream-MCPs (GitHub: OAuth 2.1+PKCE; Bitbucket: roadmap) werden vom Proxy nicht abgewickelt — Container-Setup braucht statischen Token. Token nie in User-Home, nur als Container-Secret/Env via `infrastructure-home`-Compose.

## Bekannte Limits

- **Bitbucket-Issues fehlen upstream** — der Atlassian Rovo MCP deckt fuer Bitbucket nur PRs + Pipelines ab. Issue-Workflows bleiben Web-UI. Out-of-scope upstream, nicht im Monorepo loesbar.
- **Bitbucket SSE-Endpoint** (`/v1/sse`) wird **2026-06-30** abgeschaltet — nur Streamable-HTTP `/v1/mcp` nutzen.
- **Gitea ohne hosted Endpoint** — Container muss laufen, wo `GITEA_HOST` erreichbar ist. Selbst-deployen.

## Compose-Integration

In dieser Iteration **out-of-scope** — `infrastructure-home` Compose wird vom User separat aktualisiert (analog [[aws-mcp-server]] Phase C). Aggregator-Code im Monorepo braucht keine Aenderung, Backends werden ueber `MCP_BACKENDS`-Env von compose-Seite zusammengestellt.

## Smoke

Beide Proxies (`github`, `bitbucket`) und der Gitea-Image-Build kompilieren via `docker run --rm -v $PWD:/src -w /src golang:1.26-alpine`. `scripts/verify-mcp-matrix.sh`: 16 Server konsistent.

Live-Smoke (Token noetig):

```bash
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"smoke","version":"1"}}}' \
  http://localhost:8000/
```
