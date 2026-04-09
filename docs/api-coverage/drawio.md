# draw.io MCP Proxy Coverage

- **Upstream**: [draw.io MCP App Server](https://github.com/jgraph/drawio-mcp)
- **Upstream URL**: `https://mcp.draw.io/mcp`
- **Transport**: Streamable HTTP (Proxy)
- **Auth**: Keine
- **Letzter Check**: 2026-04-09

## Tools (via Proxy)

| Tool | Parameter | Typ | Required | Beschreibung |
|------|-----------|-----|----------|-------------|
| create_diagram | xml | string | ja | draw.io XML im mxGraphModel-Format |
| search_shapes | query | string | ja | Space-separated Suchbegriffe |
| search_shapes | limit | number | nein | Max Ergebnisse (default 10, max 50) |

## Hinweise

- Kein API-Key noetig, der Upstream-Server ist oeffentlich
- Session-Management via `Mcp-Session-Id` Header (wird durchgereicht)
- `create_diagram` rendert interaktive Diagramme (Zoom, Pan, Layers) in MCP-Apps-faehigen Hosts
- `search_shapes` durchsucht 10.000+ Shapes (AWS, Azure, GCP, Cisco, Kubernetes, UML, BPMN, etc.)
