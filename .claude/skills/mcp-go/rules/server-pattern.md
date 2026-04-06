# Server Pattern

Jeder MCP-Server nutzt den Shared Bootstrap aus `internal/server/`.

## Minimales main.go

```go
package main

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/traffino/mcp-server/internal/config"
	"github.com/traffino/mcp-server/internal/server"
)

func main() {
	apiKey := config.Require("API_KEY")
	srv := server.New("server-name", "1.0.0")

	mcp.AddTool(srv.MCPServer(), &mcp.Tool{
		Name:        "tool_name",
		Description: "What this tool does",
	}, makeHandler(apiKey))

	srv.ListenAndServe(config.Get("PORT", ":8000"))
}
```

## Regeln

- Konfiguration NUR ueber Environment-Variablen (`internal/config`)
- `config.Require()` fuer Pflicht-Variablen (beendet Prozess wenn nicht gesetzt)
- `config.Get()` fuer optionale Variablen mit Default
- Port ist immer konfigurierbar, Default `:8000`
- Server-Name und Version muessen gesetzt werden
- Keine globalen Variablen — Dependencies als Closures oder Structs
