# GitHub MCP Proxy Coverage

- **Upstream**: GitHub MCP Server (offiziell von GitHub, GA seit 2025-09)
- **Upstream URL**: `https://api.githubcopilot.com/mcp/`
- **Source**: [github/github-mcp-server](https://github.com/github/github-mcp-server)
- **Transport**: Streamable HTTP (Proxy)
- **Auth**: PAT (Bearer) via `GITHUB_TOKEN`. Upstream unterstuetzt zusaetzlich OAuth 2.1 + PKCE — wird vom Proxy aktuell nicht genutzt, weil der Container-Use-Case einen statischen Token braucht.
- **Letzter Check**: 2026-05-15

## Migrations-Notiz

Frueher (2026-04-06 bis 2026-05-15) lebte in `cmd/github/` ein eigener Read-only-REST-Wrapper mit 20 Tools. Ersetzt durch diesen Proxy, weil:

- Neuer User-Scope erforderte Write-Operationen (Issue/PR create/edit/comment/review/merge) — im Eigenbau ~8-10 zusaetzliche Tools + OAuth/PAT-Handling.
- Offizieller Remote-MCP deckt Read+Write seit Sept 2025 vollstaendig ab.
- Token-Scope filtert ungewollte Tools serverseitig (kein Code-Aufwand).

## Scope (via Upstream)

Read: Repos, Issues, PRs, Actions, Releases, Search, Users, Orgs, Notifications, Projects, Code Search.
Write: Issue create/edit/comment/close+reopen/labels/assignees, PR create/edit/comment/review (approve/request-changes)/merge, Releases, Labels.

Genauer Tool-Katalog: siehe Upstream-README. Token-Scopes steuern Verfuegbarkeit (z.B. `repo`, `issues`, `pull_requests`).

## Hinweise

- Token-Wahl: Fine-grained PAT empfohlen. Scope auf benoetigte Repos + benoetigte Permissions (Issues r/w, Pull Requests r/w, Contents r, Metadata r) eingrenzen.
- Session-Management via `Mcp-Session-Id` Header (wird durchgereicht).
- Rate-Limit: GitHub-Standard (5000 req/h authenticated). Upstream-MCP zaehlt gegen das Token.
- OAuth-Flow wird vom Proxy NICHT abgewickelt — fuer Browser-OAuth muesste der Client direkt `api.githubcopilot.com/mcp/` aufrufen (an diesem Proxy vorbei).
