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
| `bitbucket` | `BITBUCKET_USER_EMAIL` + `BITBUCKET_API_TOKEN` | Atlassian Scoped API Token mit Bitbucket-Scopes (HTTP Basic, Email:Token) | Repository r/w, PR r/w, Pipeline r |
| `gitea` (extern) | `GITEA_ACCESS_TOKEN` + `GITEA_HOST` | Gitea-PAT | read/write repository + issue, plus release |

Bitbucket-Auth ist HTTP Basic mit `email:api_token`, NICHT Bearer. Bearer wird von der Bitbucket-REST-API nur fuer Workspace/Repository/Project Access Tokens und OAuth akzeptiert — bei einem Atlassian Scoped API Token liefert die API `401 "Token is invalid, expired, or not supported for this endpoint."`. Scoped API Tokens kommen aus `id.atlassian.com/manage-profile/security/api-tokens` (Atlassian-Account-weit, Scopes pro Token konfigurierbar — Bitbucket-Scopes muessen explizit ausgewaehlt sein) und loesen die per 2026-06 abgekuendigten App Passwords ab.

## Bekannte Limits

- **Bitbucket Issues + Wiki sind End-of-Life**: Atlassian stellt beide Module zum 2026-08-20 ein. Issue-Tools (`list_issues`/`issue_read`/`issue_write`) wurden deshalb 2026-05-16 aus dem Wrapper entfernt — Issue-Tracking laeuft auf Jira oder externen Trackern.
- **Gitea ohne hosted Endpoint** — Container muss laufen, wo `GITEA_HOST` erreichbar ist. Selbst-deployen via `docker.gitea.com/gitea-mcp-server`.

## Lessons Learned (2026-05-16)

**Atlassian Rovo MCP mit statischem Token ist effektiv read-only auf Teamwork-Graph** (Stand 2026-05). Die Doku-Versprechen ("PR r/w via MCP") gelten nur fuer OAuth-2.1+PKCE-Flow. Bei naechster Forge-MCP-Auswahl: vor Wrapper-Pattern in Container ein Live-Smoke gegen den Upstream mit dem geplanten Token-Typ — `tools/list`-Antwort pruefen, ob das gewuenschte Toolset wirklich exposed wird. Doku-Read alleine reicht nicht.

**Self-Build von Gitea-MCP war Overhead ohne Mehrwert** — Image vom Hersteller existiert, ist gepflegt, laeuft. Repo-Build hat beim Live-Restart Probleme gemacht (vermutlich Build-Argument-Drift gegen Upstream-Repo-Struktur — `go build .` im Root klont der Source greift nicht auf das main package, oder Flag-Parsing weicht ab). Lehre: "offizielles Image vom Hersteller existiert" → direkt in compose, nicht ueberbauen.

## Compose-Integration

`infrastructure-home` Compose pflegt User separat:

- `bitbucket`: Image `traffino/mcp-bitbucket:latest` (aus diesem Monorepo gebaut), Env `BITBUCKET_USER_EMAIL` + `BITBUCKET_API_TOKEN`
- `github`: wie gehabt (Proxy, `GITHUB_TOKEN`)
- `gitea`: direktes Image `docker.gitea.com/gitea-mcp-server` mit `command: /app/gitea-mcp -t http --port 8000`

Aggregator-Code im Monorepo braucht keine Aenderung, Backends werden ueber `MCP_BACKENDS`-Env zusammengestellt.
