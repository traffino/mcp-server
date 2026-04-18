package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SickDayAddParams struct {
	Company   string `json:"company" jsonschema:"Company name or ID"`
	StartDate string `json:"start_date" jsonschema:"First sick day (YYYY-MM-DD)"`
	EndDate   string `json:"end_date" jsonschema:"Last sick day (YYYY-MM-DD)"`
	Days      int    `json:"days" jsonschema:"Number of sick days"`
	Note      string `json:"note,omitempty" jsonschema:"Optional note"`
}

func makeSickDayAdd(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *SickDayAddParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *SickDayAddParams) (*mcp.CallToolResult, any, error) {
		if p.Company == "" || p.StartDate == "" || p.EndDate == "" || p.Days == 0 {
			return errResult("company, start_date, end_date, and days are required")
		}
		companyID, companyName, err := resolveCompany(db, p.Company)
		if err != nil {
			return errResult(err.Error())
		}
		res, err := db.Exec("INSERT INTO sick_day (company_id, start_date, end_date, days, note) VALUES (?, ?, ?, ?, ?)",
			companyID, p.StartDate, p.EndDate, p.Days, nilIfEmpty(p.Note))
		if err != nil {
			return errResult(err.Error())
		}
		id, _ := res.LastInsertId()
		return textResult(fmt.Sprintf("Added %d sick day(s) at %s: %s to %s (ID: %d)", p.Days, companyName, p.StartDate, p.EndDate, id))
	}
}

type SickDayListParams struct {
	Company string `json:"company,omitempty" jsonschema:"Filter by company name or ID"`
	Year    int    `json:"year,omitempty" jsonschema:"Filter by year (e.g. 2026)"`
}

func makeSickDayList(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *SickDayListParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *SickDayListParams) (*mcp.CallToolResult, any, error) {
		query := `SELECT s.id, c.name, s.start_date, s.end_date, s.days, s.note
			FROM sick_day s JOIN company c ON s.company_id = c.id WHERE 1=1`
		var args []any

		if p.Company != "" {
			companyID, _, err := resolveCompany(db, p.Company)
			if err != nil {
				return errResult(err.Error())
			}
			query += " AND s.company_id = ?"
			args = append(args, companyID)
		}
		if p.Year > 0 {
			query += " AND strftime('%Y', s.start_date) = ?"
			args = append(args, fmt.Sprintf("%04d", p.Year))
		}
		query += " ORDER BY s.start_date DESC"

		rows, err := db.Query(query, args...)
		if err != nil {
			return errResult(err.Error())
		}
		defer rows.Close()

		var sb strings.Builder
		sb.WriteString("Sick day entries:\n\n")
		count := 0
		for rows.Next() {
			var id int64
			var company, startDate, endDate string
			var days int
			var note sql.NullString
			rows.Scan(&id, &company, &startDate, &endDate, &days, &note)
			sb.WriteString(fmt.Sprintf("  [%d] %s to %s | %s | %d day(s)", id, startDate, endDate, company, days))
			if note.Valid {
				sb.WriteString(fmt.Sprintf(" | %s", note.String))
			}
			sb.WriteString("\n")
			count++
		}
		if count == 0 {
			return textResult("No sick day entries found.")
		}
		return textResult(sb.String())
	}
}

type SickDayUpdateParams struct {
	ID        int    `json:"id" jsonschema:"Sick day entry ID"`
	StartDate string `json:"start_date,omitempty" jsonschema:"New start date"`
	EndDate   string `json:"end_date,omitempty" jsonschema:"New end date"`
	Days      int    `json:"days,omitempty" jsonschema:"New number of days"`
	Note      string `json:"note,omitempty" jsonschema:"New note"`
}

func makeSickDayUpdate(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *SickDayUpdateParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *SickDayUpdateParams) (*mcp.CallToolResult, any, error) {
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
		if p.Note != "" {
			sets = append(sets, "note = ?")
			args = append(args, p.Note)
		}
		if len(sets) == 0 {
			return errResult("nothing to update")
		}
		args = append(args, p.ID)
		res, err := db.Exec("UPDATE sick_day SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
		if err != nil {
			return errResult(err.Error())
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return errResult(fmt.Sprintf("sick day entry with ID %d not found", p.ID))
		}
		return textResult(fmt.Sprintf("Updated sick day entry ID %d", p.ID))
	}
}

func makeSickDayDelete(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *IDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IDParam) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		res, err := db.Exec("DELETE FROM sick_day WHERE id = ?", p.ID)
		if err != nil {
			return errResult(err.Error())
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return errResult(fmt.Sprintf("sick day entry with ID %d not found", p.ID))
		}
		return textResult(fmt.Sprintf("Deleted sick day entry ID %d", p.ID))
	}
}
