package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func errResult(msg string) (*mcp.CallToolResult, any, error) {
	r := &mcp.CallToolResult{}
	r.SetError(fmt.Errorf("%s", msg))
	return r, nil, nil
}

func textResult(text string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil, nil
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nilIfZero(f float64) any {
	if f == 0 {
		return nil
	}
	return f
}

func nilIfZeroInt(i int) any {
	if i == 0 {
		return nil
	}
	return i
}

// resolveCompany resolves a company by name (case-insensitive) or ID.
func resolveCompany(db *sql.DB, input string) (int64, string, error) {
	if id, err := strconv.ParseInt(input, 10, 64); err == nil {
		var name string
		err := db.QueryRow("SELECT name FROM company WHERE id = ?", id).Scan(&name)
		if err == sql.ErrNoRows {
			return 0, "", fmt.Errorf("company with ID %d not found", id)
		}
		if err != nil {
			return 0, "", err
		}
		return id, name, nil
	}

	var id int64
	var name string
	err := db.QueryRow("SELECT id, name FROM company WHERE lower(name) = lower(?)", input).Scan(&id, &name)
	if err == sql.ErrNoRows {
		rows, _ := db.Query("SELECT name FROM company ORDER BY name")
		var names []string
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var n string
				rows.Scan(&n)
				names = append(names, n)
			}
		}
		if len(names) > 0 {
			return 0, "", fmt.Errorf("company %q not found. Known companies: %s", input, strings.Join(names, ", "))
		}
		return 0, "", fmt.Errorf("company %q not found. No companies registered yet", input)
	}
	if err != nil {
		return 0, "", err
	}
	return id, name, nil
}
