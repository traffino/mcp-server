# AWS Knowledge MCP Proxy Coverage

- **Upstream**: AWS Knowledge MCP Server (gehostet von AWS)
- **Upstream URL**: `https://knowledge-mcp.global.api.aws`
- **Transport**: Streamable HTTP (Proxy)
- **Auth**: Keine (anonym, Rate-Limited)
- **Letzter Check**: 2026-05-12

## Abgrenzung zum `aws`-Server

- `aws-docs` (dieser Server) — **Wissensquelle**: AWS-Doku, What's New, Best Practices, CDK/CloudFormation-Guidance, Strands Agents SDK. Kein Account-Zugriff.
- `aws` — **Account-Zugriff**: EC2/S3/IAM/... read-only mit eigenen IAM-Keys. Kein Doku-Wissen.

Typischer Flow: Konzept-Frage → `aws-docs`. Inventar/Audit eines konkreten Resources → `aws`.

## Tools (via Proxy)

| Tool | Beschreibung |
|------|-------------|
| `search_documentation` | AWS-Doku-Suche mit optionalen Topic-Filtern |
| `read_documentation` | Doku-Seite zu Markdown konvertieren |
| `recommend` | Content-Empfehlungen ("weiter lesen"-Hinweise) |
| `list_regions` | AWS-Regionen auflisten |
| `get_regional_availability` | Service-/API-/CloudFormation-Resource-Verfuegbarkeit pro Region |
| `retrieve_skill` | Domain-spezifische Agent-Skills (z.B. Strands Agents) |

## Hinweise

- Kein API-Key noetig, der Upstream-Server ist oeffentlich.
- Session-Management via `Mcp-Session-Id` Header (wird durchgereicht).
- Indexiert: AWS-Dokumentation, What's New Posts, Best Practices, CDK/CloudFormation, Strands Agents SDK.
- Rate-Limits sind nicht oeffentlich dokumentiert — bei haeufigen Calls auf 429 achten.
