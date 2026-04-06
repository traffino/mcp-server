# Tool Definition

Tools werden mit dem offiziellen SDK (`modelcontextprotocol/go-sdk`) registriert.

## Typed Parameters via Struct

```go
type SearchParams struct {
	Query string `json:"query" jsonschema:"Search query string"`
	Count int    `json:"count,omitempty" jsonschema:"Number of results (default 10)"`
}

func searchHandler(ctx context.Context, req *mcp.CallToolRequest, params *SearchParams) (*mcp.CallToolResult, any, error) {
	// params ist automatisch geparst und validiert
	results, err := doSearch(params.Query, params.Count)
	if err != nil {
		return mcp.NewError(mcp.InternalError, err.Error()), nil, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: formatResults(results)},
		},
	}, nil, nil
}
```

## Registrierung

```go
mcp.AddTool(srv.MCPServer(), &mcp.Tool{
	Name:        "web_search",
	Description: "Search the web using Brave Search",
}, searchHandler)
```

## Regeln

- Parameter-Structs mit `json` und `jsonschema` Tags
- `jsonschema` Tag ist die Beschreibung fuer den LLM-Client
- Optional-Felder mit `omitempty`
- Fehler als `mcp.NewError()` zurueckgeben, nicht als Go-Error (dritter Return-Wert)
- Go-Error (dritter Return-Wert) nur fuer unerwartete/interne Fehler
- Ergebnisse als `TextContent` mit JSON oder formatiertem Text
- Fuer grosse Ergebnisse: JSON bevorzugen (LLM kann es gut parsen)
