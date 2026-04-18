package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CompanyCreateParams struct {
	Name               string  `json:"name" jsonschema:"Company name"`
	WeeklyHours        float64 `json:"weekly_hours,omitempty" jsonschema:"Contractual weekly hours (e.g. 40)"`
	AnnualVacationDays int     `json:"annual_vacation_days,omitempty" jsonschema:"Annual vacation day allowance"`
}

func makeCompanyCreate(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *CompanyCreateParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *CompanyCreateParams) (*mcp.CallToolResult, any, error) {
		if p.Name == "" {
			return errResult("name is required")
		}
		res, err := db.Exec("INSERT INTO company (name, weekly_hours, annual_vacation_days) VALUES (?, ?, ?)",
			p.Name, nilIfZero(p.WeeklyHours), nilIfZeroInt(p.AnnualVacationDays))
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return errResult(fmt.Sprintf("company %q already exists", p.Name))
			}
			return errResult(err.Error())
		}
		id, _ := res.LastInsertId()
		return textResult(fmt.Sprintf("Created company %q (ID: %d)", p.Name, id))
	}
}

func makeCompanyList(db *sql.DB) func(context.Context, *mcp.CallToolRequest, any) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
		rows, err := db.Query("SELECT id, name, weekly_hours, annual_vacation_days FROM company ORDER BY name")
		if err != nil {
			return errResult(err.Error())
		}
		defer rows.Close()

		var sb strings.Builder
		sb.WriteString("Companies:\n\n")
		count := 0
		for rows.Next() {
			var id int64
			var name string
			var hours sql.NullFloat64
			var days sql.NullInt64
			rows.Scan(&id, &name, &hours, &days)
			sb.WriteString(fmt.Sprintf("  [%d] %s", id, name))
			if hours.Valid {
				sb.WriteString(fmt.Sprintf(" — %.1fh/week", hours.Float64))
			}
			if days.Valid {
				sb.WriteString(fmt.Sprintf(", %d vacation days/year", days.Int64))
			}
			sb.WriteString("\n")
			count++
		}
		if count == 0 {
			return textResult("No companies registered yet.")
		}
		return textResult(sb.String())
	}
}

type CompanyUpdateParams struct {
	ID                 int     `json:"id" jsonschema:"Company ID"`
	Name               string  `json:"name,omitempty" jsonschema:"New company name"`
	WeeklyHours        float64 `json:"weekly_hours,omitempty" jsonschema:"New weekly hours"`
	AnnualVacationDays int     `json:"annual_vacation_days,omitempty" jsonschema:"New annual vacation days"`
}

func makeCompanyUpdate(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *CompanyUpdateParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *CompanyUpdateParams) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		var sets []string
		var args []any
		if p.Name != "" {
			sets = append(sets, "name = ?")
			args = append(args, p.Name)
		}
		if p.WeeklyHours > 0 {
			sets = append(sets, "weekly_hours = ?")
			args = append(args, p.WeeklyHours)
		}
		if p.AnnualVacationDays > 0 {
			sets = append(sets, "annual_vacation_days = ?")
			args = append(args, p.AnnualVacationDays)
		}
		if len(sets) == 0 {
			return errResult("nothing to update — provide name, weekly_hours, or annual_vacation_days")
		}
		args = append(args, p.ID)
		res, err := db.Exec("UPDATE company SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
		if err != nil {
			return errResult(err.Error())
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return errResult(fmt.Sprintf("company with ID %d not found", p.ID))
		}
		return textResult(fmt.Sprintf("Updated company ID %d", p.ID))
	}
}

type CompanyDeleteParams struct {
	ID int `json:"id" jsonschema:"Company ID to delete"`
}

func makeCompanyDelete(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *CompanyDeleteParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *CompanyDeleteParams) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		var name string
		err := db.QueryRow("SELECT name FROM company WHERE id = ?", p.ID).Scan(&name)
		if err == sql.ErrNoRows {
			return errResult(fmt.Sprintf("company with ID %d not found", p.ID))
		}
		if err != nil {
			return errResult(err.Error())
		}
		_, err = db.Exec("DELETE FROM company WHERE id = ?", p.ID)
		if err != nil {
			return errResult(err.Error())
		}
		return textResult(fmt.Sprintf("Deleted company %q (ID: %d) and all its entries", name, p.ID))
	}
}
