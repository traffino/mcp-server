---
date: 2026-05-16
type: concept
tags: [mcp, forge, github, gitea, bitbucket, server, architecture]
---

# Forge-MCP-Wrapper

Drei Forge-MCP-Server (GitHub, Gitea, Bitbucket). Read + Write fuer Issues und PRs (inkl. PR-Review/Merge). Loest den vorherigen Eigenbau-GitHub-Server (Read-only, 20 Tools) ab.

## Pattern-Differenzierung (aktuell)

| Server | Pattern | Im Monorepo | Begruendung |
|---|---|---|---|
| `github` | **HTTP-Proxy** zu `api.githubcopilot.com/mcp/` | `cmd/github/`, `docker/github.Dockerfile` | Hosted Remote-MCP, Bearer-Token reicht — kein OAuth-Flow noetig. Volles Toolset (~40 Tools) wird vom Upstream geliefert. |
| `bitbucket` | **Eigenbau** gegen Bitbucket Cloud REST API 2.0 | `cmd/bitbucket/`, `docker/bitbucket.Dockerfile` | Atlassian Rovo MCP liefert mit statischem Token nur Teamwork-Graph-Tools (2 Tools), volle PR/Repo-Tools nur via OAuth. REST-API direkt ist self-pflegbar (19 Tools). |
| `gitea` | **Externes Image direkt in compose** | NICHT im Monorepo | Offizielles `docker.gitea.com/gitea-mcp-server` wird direkt eingebunden, kein Self-Build. |

## Doktrin

**Eigenbau wenn**: kein offizieller hosted Remote-MCP existiert ODER der existierende nur mit OAuth volles Toolset liefert (Token-only ≠ volles Toolset).

**Wrapper-Proxy wenn**: offizieller hosted Remote-MCP existiert UND mit statischem Token volles Toolset exposed (wie GitHub Copilot MCP, aws-docs Knowledge MCP, Cloudflare).

**Externes Image direkt einbinden wenn**: offizielles Image vom Forge-Hersteller selbst gepflegt, kein Self-Build noetig (wie Gitea).

[[aws-mcp-server]] hatte "Phase B (verworfen) — Python-Wrapper" weil das kein hosted MCP war und Pflege ohne Gegenwert dupliziert haette. Gleiche Logik fuer Bitbucket Rovo MCP — nur dass dort Pflege durch Atlassian existiert, aber OAuth-Gate die Praxistauglichkeit fuer Container-Setup blockiert.

## Auth

| Server | Env-Var | Token-Typ | Pflichtige Scopes |
|---|---|---|---|
| `github` | `GITHUB_TOKEN` | Fine-grained PAT empfohlen | Issues r/w, Pull Requests r/w, Contents r, Metadata r |
| `bitbucket` | `BITBUCKET_API_TOKEN` | Bitbucket Scoped API Token (Bearer) | Repository r/w, PR r/w, Issue r/w, Pipeline r |
| `gitea` (extern) | `GITEA_ACCESS_TOKEN` + `GITEA_HOST` | Gitea-PAT | read/write repository + issue, plus release |

Bitbucket-Cloud-Scoped-API-Tokens loesen die per 2026-06 abgekuendigten App Passwords ab. Atlassian-Account-API-Tokens (`ATLASSIAN_API_TOKEN`) sind eine andere Sorte und nicht passend.

## Bekannte Limits

- **Bitbucket-Issues**: REST-Endpoint existiert (`/repositories/{ws}/{repo}/issues`), erfordert aber das Issue-Modul im Repo aktiviert (Repo-Settings → Issues enable). 404 sonst.
- **Gitea ohne hosted Endpoint** — Container muss laufen, wo `GITEA_HOST` erreichbar ist. Selbst-deployen via `docker.gitea.com/gitea-mcp-server`.

## Lessons Learned (2026-05-16)

**Atlassian Rovo MCP mit statischem Token ist effektiv read-only auf Teamwork-Graph** (Stand 2026-05). Die Doku-Versprechen ("PR r/w via MCP") gelten nur fuer OAuth-2.1+PKCE-Flow. Bei naechster Forge-MCP-Auswahl: vor Wrapper-Pattern in Container ein Live-Smoke gegen den Upstream mit dem geplanten Token-Typ — `tools/list`-Antwort pruefen, ob das gewuenschte Toolset wirklich exposed wird. Doku-Read alleine reicht nicht.

**Self-Build von Gitea-MCP war Overhead ohne Mehrwert** — Image vom Hersteller existiert, ist gepflegt, laeuft. Repo-Build hat beim Live-Restart Probleme gemacht (vermutlich Build-Argument-Drift gegen Upstream-Repo-Struktur — `go build .` im Root klont der Source greift nicht auf das main package, oder Flag-Parsing weicht ab). Lehre: "offizielles Image vom Hersteller existiert" → direkt in compose, nicht ueberbauen.

## Compose-Integration

`infrastructure-home` Compose pflegt User separat:

- `bitbucket`: Image `traffino/mcp-bitbucket:latest` (aus diesem Monorepo gebaut), Env `BITBUCKET_API_TOKEN`
- `github`: wie gehabt (Proxy, `GITHUB_TOKEN`)
- `gitea`: direktes Image `docker.gitea.com/gitea-mcp-server` mit `command: /app/gitea-mcp -t http --port 8000`

Aggregator-Code im Monorepo braucht keine Aenderung, Backends werden ueber `MCP_BACKENDS`-Env zusammengestellt.
