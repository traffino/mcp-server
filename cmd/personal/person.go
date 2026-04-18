package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type PersonAddParams struct {
	Name string `json:"name" jsonschema:"Person name (unique)"`
	Note string `json:"note,omitempty" jsonschema:"Optional note"`
}

func makePersonAdd(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *PersonAddParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *PersonAddParams) (*mcp.CallToolResult, any, error) {
		if p.Name == "" {
			return errResult("name is required")
		}
		res, err := db.Exec("INSERT INTO person (name, note) VALUES (?, ?)", p.Name, nilIfEmpty(p.Note))
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return errResult(fmt.Sprintf("person %q already exists", p.Name))
			}
			return errResult(err.Error())
		}
		id, _ := res.LastInsertId()
		return textResult(fmt.Sprintf("Added person %q (ID: %d)", p.Name, id))
	}
}

type PersonListParams struct {
	Search string `json:"search,omitempty" jsonschema:"Case-insensitive name substring filter"`
}

func makePersonList(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *PersonListParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *PersonListParams) (*mcp.CallToolResult, any, error) {
		query := "SELECT id, name, note FROM person WHERE 1=1"
		var args []any
		if p.Search != "" {
			query += " AND lower(name) LIKE ?"
			args = append(args, "%"+strings.ToLower(p.Search)+"%")
		}
		query += " ORDER BY name"
		rows, err := db.Query(query, args...)
		if err != nil {
			return errResult(err.Error())
		}
		defer rows.Close()

		var sb strings.Builder
		sb.WriteString("People:\n\n")
		count := 0
		for rows.Next() {
			var id int64
			var name string
			var note sql.NullString
			rows.Scan(&id, &name, &note)
			sb.WriteString(fmt.Sprintf("  [%d] %s", id, name))
			if note.Valid {
				sb.WriteString(fmt.Sprintf(" — %s", note.String))
			}
			sb.WriteString("\n")
			count++
		}
		if count == 0 {
			return textResult("No people found.")
		}
		return textResult(sb.String())
	}
}

type PersonUpdateParams struct {
	ID   int    `json:"id" jsonschema:"Person ID"`
	Name string `json:"name,omitempty" jsonschema:"New name"`
	Note string `json:"note,omitempty" jsonschema:"New note"`
}

func makePersonUpdate(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *PersonUpdateParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *PersonUpdateParams) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		var sets []string
		var args []any
		if p.Name != "" {
			sets = append(sets, "name = ?")
			args = append(args, p.Name)
		}
		if p.Note != "" {
			sets = append(sets, "note = ?")
			args = append(args, p.Note)
		}
		if len(sets) == 0 {
			return errResult("nothing to update")
		}
		args = append(args, p.ID)
		res, err := db.Exec("UPDATE person SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
		if err != nil {
			return errResult(err.Error())
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return errResult(fmt.Sprintf("person with ID %d not found", p.ID))
		}
		return textResult(fmt.Sprintf("Updated person ID %d", p.ID))
	}
}

func makePersonDelete(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *IDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IDParam) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		var name string
		err := db.QueryRow("SELECT name FROM person WHERE id = ?", p.ID).Scan(&name)
		if err == sql.ErrNoRows {
			return errResult(fmt.Sprintf("person with ID %d not found", p.ID))
		}
		if err != nil {
			return errResult(err.Error())
		}
		_, err = db.Exec("DELETE FROM person WHERE id = ?", p.ID)
		if err != nil {
			return errResult(err.Error())
		}
		return textResult(fmt.Sprintf("Deleted person %q (ID: %d) and all their events", name, p.ID))
	}
}
