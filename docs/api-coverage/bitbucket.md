# Bitbucket Cloud MCP Coverage

- **Upstream**: Bitbucket Cloud REST API 2.0 (Eigenbau-Wrapper, kein Atlassian-MCP)
- **Upstream URL**: `https://api.bitbucket.org/2.0`
- **Doku**: [Bitbucket Cloud REST API](https://developer.atlassian.com/cloud/bitbucket/rest/intro/)
- **Auth**: Scoped API Token (Bitbucket-spezifisch) als Bearer-Token via `BITBUCKET_API_TOKEN`. **Keine App Passwords** — werden Juni 2026 abgekuendigt.
- **Tenancy**: Cloud-only. Workspace: `baltasaar` (bitbucket.org/baltasaar).
- **Letzter Check**: 2026-05-16

## Tools (19)

### Read

| Tool | Endpoint |
|---|---|
| `get_me` | `GET /user` |
| `list_workspaces` | `GET /workspaces` |
| `list_repositories` | `GET /repositories/{workspace}` |
| `get_repository` | `GET /repositories/{workspace}/{repo_slug}` |
| `get_file_contents` | `GET /repositories/{workspace}/{repo_slug}/src/{ref}/{path}` |
| `list_branches` | `GET /repositories/{workspace}/{repo_slug}/refs/branches` |
| `list_commits` | `GET /repositories/{workspace}/{repo_slug}/commits` |
| `get_commit` | `GET /repositories/{workspace}/{repo_slug}/commit/{hash}` |
| `list_pull_requests` | `GET /repositories/{workspace}/{repo_slug}/pullrequests?state=...` |
| `pull_request_read` | `GET /repositories/{workspace}/{repo_slug}/pullrequests/{id}` plus optional `/diff`, `/activity`, `/commits`, `/statuses` |
| `list_pipelines` | `GET /repositories/{workspace}/{repo_slug}/pipelines/` |
| `get_pipeline` | `GET /repositories/{workspace}/{repo_slug}/pipelines/{uuid}` + optional `/steps/` |
| `list_issues` | `GET /repositories/{workspace}/{repo_slug}/issues` |
| `issue_read` | `GET /repositories/{workspace}/{repo_slug}/issues/{id}` + optional `/comments` |

### Write

| Tool | Operation |
|---|---|
| `pull_request_write` | `action=create`: `POST /pullrequests` · `action=update`: `PUT /pullrequests/{id}` |
| `pull_request_review_write` | `action=approve|unapprove|request-changes|unrequest-changes`: `POST/DELETE /pullrequests/{id}/approve\|request-changes` · `action=decline`: `POST /pullrequests/{id}/decline` |
| `pull_request_comment_write` | `POST /pullrequests/{id}/comments` (raw + optional inline path/line) |
| `merge_pull_request` | `POST /pullrequests/{id}/merge` (merge_commit / squash / fast_forward) |
| `issue_write` | `action=create`: `POST /issues` · `action=update`: `PUT /issues/{id}` · `action=comment`: `POST /issues/{id}/comments` |

## Hinweise

- **Issues** erfordern dass das Issue-Modul im Repo aktiviert ist (Repo-Settings). 404 falls deaktiviert.
- **Pagination**: `pagelen` (max 100) und `page` (1-indexed). Default je nach Tool 25-50.
- **BBQL** unterstuetzt fuer `q`-Parameter (z.B. `state = "OPEN" AND author.username = "x"`). Doku: Atlassian BBQL.
- Antworten kommen als formatiertes JSON zurueck (pretty-printed). Bei `pull_request_read` mit `include=diff` als Plain-Text-Diff.
- Tool-Praefix in Aggregator: `bitbucket_*`.

## Vorgeschichte

Erste Iteration (PR #5, 2026-05-15) war HTTP-Proxy zu `mcp.atlassian.com/v1/mcp` (Atlassian Rovo MCP). Mit statischem API-Token expose der Rovo-MCP nur zwei generische `getTeamworkGraph*`-Tools — kein PR/Repo-Toolset. Das volle Toolset verlangt OAuth-2.1+PKCE, was im Proxy-Stil nicht abbildbar ist. Daher: Eigenbau gegen die REST-API direkt.
