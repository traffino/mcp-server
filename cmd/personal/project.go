package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ProjectAddParams struct {
	Name    string `json:"name" jsonschema:"Project name"`
	Company string `json:"company,omitempty" jsonschema:"Company name or ID; omit for private project"`
	Note    string `json:"note,omitempty" jsonschema:"Optional note"`
}

func makeProjectAdd(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *ProjectAddParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ProjectAddParams) (*mcp.CallToolResult, any, error) {
		if p.Name == "" {
			return errResult("name is required")
		}
		var companyID any = nil
		scope := "private"
		if p.Company != "" {
			id, name, err := resolveCompany(db, p.Company)
			if err != nil {
				return errResult(err.Error())
			}
			companyID = id
			scope = name
		}
		res, err := db.Exec(
			"INSERT INTO project (name, company_id, note) VALUES (?, ?, ?)",
			p.Name, companyID, nilIfEmpty(p.Note),
		)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return errResult(fmt.Sprintf("project %q already exists in %s", p.Name, scope))
			}
			return errResult(err.Error())
		}
		id, _ := res.LastInsertId()
		return textResult(fmt.Sprintf("Added project %q in %s (ID: %d)", p.Name, scope, id))
	}
}

type ProjectListParams struct {
	Company string `json:"company,omitempty" jsonschema:"Filter by company name/ID (use 'private' for private projects)"`
	Status  string `json:"status,omitempty" jsonschema:"Filter by status: active (default), archived, or all"`
}

func makeProjectList(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *ProjectListParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ProjectListParams) (*mcp.CallToolResult, any, error) {
		query := `SELECT p.id, p.name, COALESCE(c.name, '-'), p.status, p.note
			FROM project p LEFT JOIN company c ON p.company_id = c.id WHERE 1=1`
		var args []any

		if p.Company != "" {
			if strings.ToLower(p.Company) == "private" || p.Company == "-" {
				query += " AND p.company_id IS NULL"
			} else {
				companyID, _, err := resolveCompany(db, p.Company)
				if err != nil {
					return errResult(err.Error())
				}
				query += " AND p.company_id = ?"
				args = append(args, companyID)
			}
		}
		status := p.Status
		if status == "" {
			status = "active"
		}
		if status != "all" {
			if status != "active" && status != "archived" {
				return errResult("status must be one of: active, archived, all")
			}
			query += " AND p.status = ?"
			args = append(args, status)
		}
		query += " ORDER BY COALESCE(c.name, '-'), p.name"

		rows, err := db.Query(query, args...)
		if err != nil {
			return errResult(err.Error())
		}
		defer rows.Close()

		var sb strings.Builder
		sb.WriteString("Projects:\n\n")
		count := 0
		for rows.Next() {
			var id int64
			var name, company, st string
			var note sql.NullString
			rows.Scan(&id, &name, &company, &st, &note)
			sb.WriteString(fmt.Sprintf("  [%d] %s | %s | %s", id, company, name, st))
			if note.Valid {
				sb.WriteString(fmt.Sprintf(" | %s", note.String))
			}
			sb.WriteString("\n")
			count++
		}
		if count == 0 {
			return textResult("No projects found.")
		}
		return textResult(sb.String())
	}
}

type ProjectUpdateParams struct {
	ID      int    `json:"id" jsonschema:"Project ID"`
	Name    string `json:"name,omitempty" jsonschema:"New name"`
	Company string `json:"company,omitempty" jsonschema:"New company (name/ID) or 'private' to detach"`
	Status  string `json:"status,omitempty" jsonschema:"New status: active or archived"`
	Note    string `json:"note,omitempty" jsonschema:"New note"`
}

func makeProjectUpdate(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *ProjectUpdateParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ProjectUpdateParams) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		var sets []string
		var args []any
		if p.Name != "" {
			sets = append(sets, "name = ?")
			args = append(args, p.Name)
		}
		if p.Company != "" {
			if strings.ToLower(p.Company) == "private" || p.Company == "-" {
				sets = append(sets, "company_id = NULL")
			} else {
				cid, _, err := resolveCompany(db, p.Company)
				if err != nil {
					return errResult(err.Error())
				}
				sets = append(sets, "company_id = ?")
				args = append(args, cid)
			}
		}
		if p.Status != "" {
			if p.Status != "active" && p.Status != "archived" {
				return errResult("status must be active or archived")
			}
			sets = append(sets, "status = ?")
			args = append(args, p.Status)
		}
		if p.Note != "" {
			sets = append(sets, "note = ?")
			args = append(args, p.Note)
		}
		if len(sets) == 0 {
			return errResult("nothing to update")
		}
		args = append(args, p.ID)
		res, err := db.Exec("UPDATE project SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
		if err != nil {
			return errResult(err.Error())
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return errResult(fmt.Sprintf("project with ID %d not found", p.ID))
		}
		return textResult(fmt.Sprintf("Updated project ID %d", p.ID))
	}
}

func makeProjectArchive(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *IDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IDParam) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		res, err := db.Exec("UPDATE project SET status = 'archived' WHERE id = ?", p.ID)
		if err != nil {
			return errResult(err.Error())
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return errResult(fmt.Sprintf("project with ID %d not found", p.ID))
		}
		return textResult(fmt.Sprintf("Archived project ID %d", p.ID))
	}
}

func makeProjectDelete(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *IDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IDParam) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		var name string
		err := db.QueryRow("SELECT name FROM project WHERE id = ?", p.ID).Scan(&name)
		if err == sql.ErrNoRows {
			return errResult(fmt.Sprintf("project with ID %d not found", p.ID))
		}
		if err != nil {
			return errResult(err.Error())
		}
		var linkedTodos int
		db.QueryRow("SELECT COUNT(*) FROM todo WHERE project_id = ?", p.ID).Scan(&linkedTodos)

		if _, err := db.Exec("DELETE FROM project WHERE id = ?", p.ID); err != nil {
			return errResult(err.Error())
		}
		return textResult(fmt.Sprintf("Deleted project %q and %d linked todos (ID: %d)", name, linkedTodos, p.ID))
	}
}
