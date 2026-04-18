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

// resolveProject resolves a project by name (case-insensitive) or ID.
// If companyHint is non-empty, name lookups are scoped to that company (or NULL for private).
// Returns error on ambiguity.
func resolveProject(db *sql.DB, input, companyHint string) (int64, string, error) {
	if id, err := strconv.ParseInt(input, 10, 64); err == nil {
		var name string
		err := db.QueryRow("SELECT name FROM project WHERE id = ?", id).Scan(&name)
		if err == sql.ErrNoRows {
			return 0, "", fmt.Errorf("project with ID %d not found", id)
		}
		if err != nil {
			return 0, "", err
		}
		return id, name, nil
	}

	if companyHint != "" {
		if strings.ToLower(companyHint) == "private" || companyHint == "-" {
			var id int64
			var name string
			err := db.QueryRow("SELECT id, name FROM project WHERE lower(name) = lower(?) AND company_id IS NULL", input).Scan(&id, &name)
			if err == sql.ErrNoRows {
				return 0, "", fmt.Errorf("private project %q not found", input)
			}
			if err != nil {
				return 0, "", err
			}
			return id, name, nil
		}
		companyID, _, err := resolveCompany(db, companyHint)
		if err != nil {
			return 0, "", err
		}
		var id int64
		var name string
		err = db.QueryRow("SELECT id, name FROM project WHERE lower(name) = lower(?) AND company_id = ?", input, companyID).Scan(&id, &name)
		if err == sql.ErrNoRows {
			return 0, "", fmt.Errorf("project %q not found in given company", input)
		}
		if err != nil {
			return 0, "", err
		}
		return id, name, nil
	}

	rows, err := db.Query(`SELECT p.id, p.name, COALESCE(c.name, '-') FROM project p
		LEFT JOIN company c ON p.company_id = c.id
		WHERE lower(p.name) = lower(?)`, input)
	if err != nil {
		return 0, "", err
	}
	defer rows.Close()
	type match struct {
		id      int64
		name    string
		company string
	}
	var matches []match
	for rows.Next() {
		var m match
		rows.Scan(&m.id, &m.name, &m.company)
		matches = append(matches, m)
	}
	switch len(matches) {
	case 0:
		return 0, "", fmt.Errorf("project %q not found", input)
	case 1:
		return matches[0].id, matches[0].name, nil
	default:
		var companies []string
		for _, m := range matches {
			companies = append(companies, m.company)
		}
		return 0, "", fmt.Errorf("project %q is ambiguous — exists in: %s. Use ID or pass company=<name>", input, strings.Join(companies, ", "))
	}
}

// resolvePerson resolves a person by name (case-insensitive) or ID.
func resolvePerson(db *sql.DB, input string) (int64, string, error) {
	if id, err := strconv.ParseInt(input, 10, 64); err == nil {
		var name string
		err := db.QueryRow("SELECT name FROM person WHERE id = ?", id).Scan(&name)
		if err == sql.ErrNoRows {
			return 0, "", fmt.Errorf("person with ID %d not found", id)
		}
		if err != nil {
			return 0, "", err
		}
		return id, name, nil
	}
	var id int64
	var name string
	err := db.QueryRow("SELECT id, name FROM person WHERE lower(name) = lower(?)", input).Scan(&id, &name)
	if err == sql.ErrNoRows {
		return 0, "", fmt.Errorf("person %q not found", input)
	}
	if err != nil {
		return 0, "", err
	}
	return id, name, nil
}
