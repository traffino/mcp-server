# Gitea MCP Coverage

- **Upstream**: Gitea MCP Server (offiziell vom Gitea-Projekt)
- **Source**: [gitea.com/gitea/gitea-mcp](https://gitea.com/gitea/gitea-mcp)
- **Pattern**: Externes Binary (analog `memory`) — kein eigener Go-Wrapper, das Upstream-Binary spricht selbst Streamable HTTP
- **Transport**: Streamable HTTP auf `/mcp`
- **Auth**: Gitea Personal Access Token via `GITEA_ACCESS_TOKEN`
- **Gitea-Host**: `GITEA_HOST` (z.B. `https://gitea.baunach.work`)
- **Letzter Check**: 2026-05-15

## Build

Source wird beim Image-Build aus `gitea.com/gitea/gitea-mcp` geklont und gebaut. Kein `cmd/gitea/`-Verzeichnis im Monorepo (das `verify-mcp-matrix.sh`-Skript erlaubt diese Exception fuer pure-external Server).

ENTRYPOINT startet `gitea-mcp -t http --host 0.0.0.0 --port 8000`.

## Scope (via Upstream)

Read: Repos, Branches, Commits, Files, Issues, PRs, Releases, Tags, Labels, Milestones, Notifications, Users, Orgs, Actions/Workflows, Wiki, Packages, Timetracking.

Write: Issue create/edit/comment/close+reopen, Labels, Milestones, PR create/edit/comment/review (approve/request-changes/comment), Releases, Tags, Branches, Files, Repos, Notifications, Wiki, Timetracking.

Tool-Praefix in Aggregator: `gitea_*`. Genauer Tool-Katalog: siehe Upstream-README (v1.3.0+ ~45 Tools).

## Hinweise

- PAT-Scopes in Gitea-UI: Mindestens `read:repository`, `write:repository`, `read:issue`, `write:issue`. Fuer Releases zusaetzlich `write:repository`.
- Token niemals in User-Home — nur als Container-Secret/Env. Konfig in `infrastructure-home`-Compose.
- Self-hosted: kein zentraler Remote-Endpoint, Image muss laufen wo `GITEA_HOST` erreichbar ist.
