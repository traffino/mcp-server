# Bitbucket MCP Proxy Coverage

- **Upstream**: Atlassian Rovo MCP Server (offiziell von Atlassian)
- **Upstream URL**: `https://mcp.atlassian.com/v1/mcp`
- **Doku**: [Atlassian Rovo MCP — Bitbucket Cloud](https://support.atlassian.com/bitbucket-cloud/docs/interacting-with-bitbucket-via-mcp/)
- **Transport**: Streamable HTTP (Proxy). Der alte SSE-Endpoint `/v1/sse` wird per **2026-06-30** abgeschaltet.
- **Auth**: Scoped API Token via `ATLASSIAN_API_TOKEN` (Bearer). **Keine App Passwords** — werden Juni 2026 abgekuendigt.
- **Tenancy**: Cloud-only (kein Data Center). Workspace: `baltasaar` (bitbucket.org/baltasaar).
- **Letzter Check**: 2026-05-15

## Scope (via Upstream)

Bitbucket-spezifische Tool-Familie `bitbucketPullRequest*` + Pipelines.

Read: PR list/get/diff/files, PR comments, PR tasks, PR commit-status, Pipelines.

Write:
- `createPullRequest`, `updatePullRequest`
- `addPullRequestComment`
- `approvePullRequest`, `unapprovePullRequest`
- `mergePullRequest`, `declinePullRequest`
- `createPullRequestTask`, `updatePullRequestTask`

## Bekannte Limits

- **Bitbucket-Issues fehlen komplett** — der Atlassian Rovo MCP deckt nur PRs + Pipelines fuer Bitbucket ab. Issue-Workflows muessen ueber die Web-UI laufen. Out-of-scope upstream, nicht im Monorepo loesbar.
- **OAuth ist "on roadmap"**, noch nicht verfuegbar — API-Token-only. Im Container-Secret/Env haltbar, nie in User-Home.
- App Passwords sind tot ab Juni 2026 — beim Setup nur Scoped API Token erstellen.
- Admin muss API-Token-Auth in den Rovo-MCP-Settings explizit aktivieren (Atlassian-Org-Setting).

## Hinweise

- Session-Management via `Mcp-Session-Id` Header (wird durchgereicht).
- Tool-Praefix in Aggregator: `bitbucket_*`.
- Rate-Limits: Atlassian Cloud-Standard, an das Token gebunden.
