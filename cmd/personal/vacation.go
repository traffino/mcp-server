package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type VacationAddParams struct {
	Company   string  `json:"company" jsonschema:"Company name or ID"`
	StartDate string  `json:"start_date" jsonschema:"First vacation day (YYYY-MM-DD)"`
	EndDate   string  `json:"end_date" jsonschema:"Last vacation day (YYYY-MM-DD)"`
	Days      float64 `json:"days" jsonschema:"Number of vacation days (supports half days, e.g. 0.5)"`
	Type      string  `json:"type,omitempty" jsonschema:"Type: vacation (default) or special_leave"`
	Note      string  `json:"note,omitempty" jsonschema:"Optional note"`
}

func makeVacationAdd(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *VacationAddParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *VacationAddParams) (*mcp.CallToolResult, any, error) {
		if p.Company == "" || p.StartDate == "" || p.EndDate == "" || p.Days == 0 {
			return errResult("company, start_date, end_date, and days are required")
		}
		companyID, companyName, err := resolveCompany(db, p.Company)
		if err != nil {
			return errResult(err.Error())
		}
		vType := "vacation"
		if p.Type != "" {
			vType = p.Type
		}
		res, err := db.Exec("INSERT INTO vacation (company_id, start_date, end_date, days, type, note) VALUES (?, ?, ?, ?, ?, ?)",
			companyID, p.StartDate, p.EndDate, p.Days, vType, nilIfEmpty(p.Note))
		if err != nil {
			return errResult(err.Error())
		}
		id, _ := res.LastInsertId()
		return textResult(fmt.Sprintf("Added %.1f %s day(s) at %s: %s to %s (ID: %d)", p.Days, vType, companyName, p.StartDate, p.EndDate, id))
	}
}

type VacationListParams struct {
	Company string `json:"company,omitempty" jsonschema:"Filter by company name or ID"`
	Year    int    `json:"year,omitempty" jsonschema:"Filter by year (e.g. 2026)"`
}

func makeVacationList(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *VacationListParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *VacationListParams) (*mcp.CallToolResult, any, error) {
		query := `SELECT v.id, c.name, v.start_date, v.end_date, v.days, v.type, v.note
			FROM vacation v JOIN company c ON v.company_id = c.id WHERE 1=1`
		var args []any

		if p.Company != "" {
			companyID, _, err := resolveCompany(db, p.Company)
			if err != nil {
				return errResult(err.Error())
			}
			query += " AND v.company_id = ?"
			args = append(args, companyID)
		}
		if p.Year > 0 {
			query += " AND strftime('%Y', v.start_date) = ?"
			args = append(args, fmt.Sprintf("%04d", p.Year))
		}
		query += " ORDER BY v.start_date DESC"

		rows, err := db.Query(query, args...)
		if err != nil {
			return errResult(err.Error())
		}
		defer rows.Close()

		var sb strings.Builder
		sb.WriteString("Vacation entries:\n\n")
		count := 0
		for rows.Next() {
			var id int64
			var company, startDate, endDate, vType string
			var days float64
			var note sql.NullString
			rows.Scan(&id, &company, &startDate, &endDate, &days, &vType, &note)
			sb.WriteString(fmt.Sprintf("  [%d] %s to %s | %s | %.1f days (%s)", id, startDate, endDate, company, days, vType))
			if note.Valid {
				sb.WriteString(fmt.Sprintf(" | %s", note.String))
			}
			sb.WriteString("\n")
			count++
		}
		if count == 0 {
			return textResult("No vacation entries found.")
		}
		return textResult(sb.String())
	}
}

type VacationUpdateParams struct {
	ID        int     `json:"id" jsonschema:"Vacation entry ID"`
	StartDate string  `json:"start_date,omitempty" jsonschema:"New start date"`
	EndDate   string  `json:"end_date,omitempty" jsonschema:"New end date"`
	Days      float64 `json:"days,omitempty" jsonschema:"New number of days"`
	Type      string  `json:"type,omitempty" jsonschema:"New type (vacation or special_leave)"`
	Note      string  `json:"note,omitempty" jsonschema:"New note"`
}

func makeVacationUpdate(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *VacationUpdateParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *VacationUpdateParams) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		var sets []string
		var args []any
		if p.StartDate != "" {
			sets = append(sets, "start_date = ?")
			args = append(args, p.StartDate)
		}
		if p.EndDate != "" {
			sets = append(sets, "end_date = ?")
			args = append(args, p.EndDate)
		}
		if p.Days != 0 {
			sets = append(sets, "days = ?")
			args = append(args, p.Days)
		}
		if p.Type != "" {
			sets = append(sets, "type = ?")
			args = append(args, p.Type)
		}
		if p.Note != "" {
			sets = append(sets, "note = ?")
			args = append(args, p.Note)
		}
		if len(sets) == 0 {
			return errResult("nothing to update")
		}
		args = append(args, p.ID)
		res, err := db.Exec("UPDATE vacation SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
		if err != nil {
			return errResult(err.Error())
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return errResult(fmt.Sprintf("vacation entry with ID %d not found", p.ID))
		}
		return textResult(fmt.Sprintf("Updated vacation entry ID %d", p.ID))
	}
}

func makeVacationDelete(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *IDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IDParam) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		res, err := db.Exec("DELETE FROM vacation WHERE id = ?", p.ID)
		if err != nil {
			return errResult(err.Error())
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return errResult(fmt.Sprintf("vacation entry with ID %d not found", p.ID))
		}
		return textResult(fmt.Sprintf("Deleted vacation entry ID %d", p.ID))
	}
}

type VacationBalanceParams struct {
	Company string `json:"company,omitempty" jsonschema:"Filter by company name or ID"`
	Year    int    `json:"year,omitempty" jsonschema:"Year to check (default: current year)"`
}

func makeVacationBalance(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *VacationBalanceParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *VacationBalanceParams) (*mcp.CallToolResult, any, error) {
		year := p.Year
		if year == 0 {
			year = currentYear()
		}

		query := `SELECT c.id, c.name, c.annual_vacation_days,
				COALESCE(SUM(v.days), 0) AS taken
			FROM company c
			LEFT JOIN vacation v ON v.company_id = c.id
				AND v.type = 'vacation'
				AND strftime('%Y', v.start_date) = ?
			WHERE 1=1`
		args := []any{fmt.Sprintf("%04d", year)}

		if p.Company != "" {
			companyID, _, err := resolveCompany(db, p.Company)
			if err != nil {
				return errResult(err.Error())
			}
			query += " AND c.id = ?"
			args = append(args, companyID)
		}
		query += " GROUP BY c.id ORDER BY c.name"

		rows, err := db.Query(query, args...)
		if err != nil {
			return errResult(err.Error())
		}
		defer rows.Close()

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Vacation Balance %d:\n\n", year))
		count := 0
		for rows.Next() {
			var id int64
			var name string
			var annualDays sql.NullInt64
			var taken float64
			rows.Scan(&id, &name, &annualDays, &taken)
			if annualDays.Valid {
				remaining := float64(annualDays.Int64) - taken
				sb.WriteString(fmt.Sprintf("  %s: %.1f / %d days taken, %.1f remaining\n", name, taken, annualDays.Int64, remaining))
			} else {
				sb.WriteString(fmt.Sprintf("  %s: %.1f days taken (no annual allowance set)\n", name, taken))
			}
			count++
		}
		if count == 0 {
			return textResult("No companies found.")
		}
		return textResult(sb.String())
	}
}
