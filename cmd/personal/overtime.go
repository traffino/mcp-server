package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type OvertimeAddParams struct {
	Company string  `json:"company" jsonschema:"Company name or ID"`
	Date    string  `json:"date" jsonschema:"Date (YYYY-MM-DD)"`
	Hours   float64 `json:"hours" jsonschema:"Overtime hours (negative for reduction)"`
	Reason  string  `json:"reason,omitempty" jsonschema:"Optional reason"`
}

func makeOvertimeAdd(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *OvertimeAddParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *OvertimeAddParams) (*mcp.CallToolResult, any, error) {
		if p.Company == "" || p.Date == "" {
			return errResult("company and date are required")
		}
		if p.Hours == 0 {
			return errResult("hours must be non-zero")
		}
		companyID, companyName, err := resolveCompany(db, p.Company)
		if err != nil {
			return errResult(err.Error())
		}
		res, err := db.Exec("INSERT INTO overtime (company_id, date, hours, reason) VALUES (?, ?, ?, ?)",
			companyID, p.Date, p.Hours, nilIfEmpty(p.Reason))
		if err != nil {
			return errResult(err.Error())
		}
		id, _ := res.LastInsertId()
		sign := "+"
		if p.Hours < 0 {
			sign = ""
		}
		return textResult(fmt.Sprintf("Added %s%.1fh overtime at %s on %s (ID: %d)", sign, p.Hours, companyName, p.Date, id))
	}
}

type OvertimeListParams struct {
	Company string `json:"company,omitempty" jsonschema:"Filter by company name or ID"`
	Year    int    `json:"year,omitempty" jsonschema:"Filter by year (e.g. 2026)"`
	Month   int    `json:"month,omitempty" jsonschema:"Filter by month (1-12)"`
}

func makeOvertimeList(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *OvertimeListParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *OvertimeListParams) (*mcp.CallToolResult, any, error) {
		query := `SELECT o.id, c.name, o.date, o.hours, o.reason
			FROM overtime o JOIN company c ON o.company_id = c.id WHERE 1=1`
		var args []any

		if p.Company != "" {
			companyID, _, err := resolveCompany(db, p.Company)
			if err != nil {
				return errResult(err.Error())
			}
			query += " AND o.company_id = ?"
			args = append(args, companyID)
		}
		if p.Year > 0 {
			query += " AND strftime('%Y', o.date) = ?"
			args = append(args, fmt.Sprintf("%04d", p.Year))
		}
		if p.Month > 0 {
			query += " AND strftime('%m', o.date) = ?"
			args = append(args, fmt.Sprintf("%02d", p.Month))
		}
		query += " ORDER BY o.date DESC"

		rows, err := db.Query(query, args...)
		if err != nil {
			return errResult(err.Error())
		}
		defer rows.Close()

		var sb strings.Builder
		sb.WriteString("Overtime entries:\n\n")
		count := 0
		for rows.Next() {
			var id int64
			var company, date string
			var hours float64
			var reason sql.NullString
			rows.Scan(&id, &company, &date, &hours, &reason)
			sb.WriteString(fmt.Sprintf("  [%d] %s | %s | %+.1fh", id, date, company, hours))
			if reason.Valid {
				sb.WriteString(fmt.Sprintf(" | %s", reason.String))
			}
			sb.WriteString("\n")
			count++
		}
		if count == 0 {
			return textResult("No overtime entries found.")
		}
		return textResult(sb.String())
	}
}

type OvertimeUpdateParams struct {
	ID     int     `json:"id" jsonschema:"Overtime entry ID"`
	Date   string  `json:"date,omitempty" jsonschema:"New date (YYYY-MM-DD)"`
	Hours  float64 `json:"hours,omitempty" jsonschema:"New hours value"`
	Reason string  `json:"reason,omitempty" jsonschema:"New reason"`
}

func makeOvertimeUpdate(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *OvertimeUpdateParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *OvertimeUpdateParams) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		var sets []string
		var args []any
		if p.Date != "" {
			sets = append(sets, "date = ?")
			args = append(args, p.Date)
		}
		if p.Hours != 0 {
			sets = append(sets, "hours = ?")
			args = append(args, p.Hours)
		}
		if p.Reason != "" {
			sets = append(sets, "reason = ?")
			args = append(args, p.Reason)
		}
		if len(sets) == 0 {
			return errResult("nothing to update")
		}
		args = append(args, p.ID)
		res, err := db.Exec("UPDATE overtime SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
		if err != nil {
			return errResult(err.Error())
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return errResult(fmt.Sprintf("overtime entry with ID %d not found", p.ID))
		}
		return textResult(fmt.Sprintf("Updated overtime entry ID %d", p.ID))
	}
}

type IDParam struct {
	ID int `json:"id" jsonschema:"Entry ID"`
}

func makeOvertimeDelete(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *IDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IDParam) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		res, err := db.Exec("DELETE FROM overtime WHERE id = ?", p.ID)
		if err != nil {
			return errResult(err.Error())
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return errResult(fmt.Sprintf("overtime entry with ID %d not found", p.ID))
		}
		return textResult(fmt.Sprintf("Deleted overtime entry ID %d", p.ID))
	}
}

type OvertimeSummaryParams struct {
	Company string `json:"company,omitempty" jsonschema:"Filter by company name or ID"`
	Year    int    `json:"year,omitempty" jsonschema:"Filter by year (e.g. 2026)"`
	Month   int    `json:"month,omitempty" jsonschema:"Filter by month (1-12)"`
}

func makeOvertimeSummary(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *OvertimeSummaryParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *OvertimeSummaryParams) (*mcp.CallToolResult, any, error) {
		query := `SELECT c.name, strftime('%Y-%m', o.date) AS month, SUM(o.hours) AS total
			FROM overtime o JOIN company c ON o.company_id = c.id WHERE 1=1`
		var args []any

		if p.Company != "" {
			companyID, _, err := resolveCompany(db, p.Company)
			if err != nil {
				return errResult(err.Error())
			}
			query += " AND o.company_id = ?"
			args = append(args, companyID)
		}
		if p.Year > 0 {
			query += " AND strftime('%Y', o.date) = ?"
			args = append(args, fmt.Sprintf("%04d", p.Year))
		}
		if p.Month > 0 {
			query += " AND strftime('%m', o.date) = ?"
			args = append(args, fmt.Sprintf("%02d", p.Month))
		}
		query += " GROUP BY c.name, month ORDER BY c.name, month"

		rows, err := db.Query(query, args...)
		if err != nil {
			return errResult(err.Error())
		}
		defer rows.Close()

		var sb strings.Builder
		sb.WriteString("Overtime Summary:\n\n")
		count := 0
		grandTotal := 0.0
		for rows.Next() {
			var company, month string
			var total float64
			rows.Scan(&company, &month, &total)
			sb.WriteString(fmt.Sprintf("  %s | %s | %+.1fh\n", company, month, total))
			grandTotal += total
			count++
		}
		if count == 0 {
			return textResult("No overtime entries found.")
		}
		sb.WriteString(fmt.Sprintf("\n  Total: %+.1fh\n", grandTotal))
		return textResult(sb.String())
	}
}
