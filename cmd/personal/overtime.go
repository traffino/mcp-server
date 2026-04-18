package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- overtime_add_work ---

type OvertimeAddWorkParams struct {
	Company   string `json:"company" jsonschema:"Company name or ID"`
	Date      string `json:"date" jsonschema:"Date (YYYY-MM-DD)"`
	StartTime string `json:"start_time" jsonschema:"Start time (HH:MM)"`
	EndTime   string `json:"end_time" jsonschema:"End time (HH:MM)"`
	Reason    string `json:"reason,omitempty" jsonschema:"Optional reason"`
}

func makeOvertimeAddWork(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *OvertimeAddWorkParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *OvertimeAddWorkParams) (*mcp.CallToolResult, any, error) {
		if p.Company == "" || p.Date == "" || p.StartTime == "" || p.EndTime == "" {
			return errResult("company, date, start_time, and end_time are required")
		}
		hours, err := ComputeHours(p.StartTime, p.EndTime)
		if err != nil {
			return errResult(err.Error())
		}
		companyID, companyName, err := resolveCompany(db, p.Company)
		if err != nil {
			return errResult(err.Error())
		}
		res, err := db.Exec(
			"INSERT INTO overtime (company_id, date, type, start_time, end_time, hours, reason) VALUES (?, ?, 'work', ?, ?, ?, ?)",
			companyID, p.Date, p.StartTime, p.EndTime, hours, nilIfEmpty(p.Reason),
		)
		if err != nil {
			return errResult(err.Error())
		}
		id, _ := res.LastInsertId()
		return textResult(fmt.Sprintf("Added %.2fh work (%s–%s) at %s on %s (ID: %d)", hours, p.StartTime, p.EndTime, companyName, p.Date, id))
	}
}

// --- overtime_add_reduction ---

type OvertimeAddReductionParams struct {
	Company string  `json:"company" jsonschema:"Company name or ID"`
	Date    string  `json:"date" jsonschema:"Date (YYYY-MM-DD)"`
	Hours   float64 `json:"hours" jsonschema:"Hours to deduct from balance (positive number)"`
	Reason  string  `json:"reason,omitempty" jsonschema:"Optional reason"`
}

func makeOvertimeAddReduction(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *OvertimeAddReductionParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *OvertimeAddReductionParams) (*mcp.CallToolResult, any, error) {
		if p.Company == "" || p.Date == "" {
			return errResult("company and date are required")
		}
		if p.Hours <= 0 {
			return errResult("hours must be positive (reductions are stored as positive, summed negatively)")
		}
		companyID, companyName, err := resolveCompany(db, p.Company)
		if err != nil {
			return errResult(err.Error())
		}
		res, err := db.Exec(
			"INSERT INTO overtime (company_id, date, type, hours, reason) VALUES (?, ?, 'reduction', ?, ?)",
			companyID, p.Date, p.Hours, nilIfEmpty(p.Reason),
		)
		if err != nil {
			return errResult(err.Error())
		}
		id, _ := res.LastInsertId()
		return textResult(fmt.Sprintf("Added -%.2fh reduction at %s on %s (ID: %d)", p.Hours, companyName, p.Date, id))
	}
}

// --- overtime_list ---

type OvertimeListParams struct {
	Company string `json:"company,omitempty" jsonschema:"Filter by company name or ID"`
	Year    int    `json:"year,omitempty" jsonschema:"Filter by year (e.g. 2026)"`
	Month   int    `json:"month,omitempty" jsonschema:"Filter by month (1-12)"`
	Type    string `json:"type,omitempty" jsonschema:"Filter by type: work or reduction"`
}

func makeOvertimeList(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *OvertimeListParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *OvertimeListParams) (*mcp.CallToolResult, any, error) {
		query := `SELECT o.id, c.name, o.date, o.type, o.start_time, o.end_time, o.hours, o.reason
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
		if p.Type != "" {
			if p.Type != "work" && p.Type != "reduction" {
				return errResult("type must be 'work' or 'reduction'")
			}
			query += " AND o.type = ?"
			args = append(args, p.Type)
		}
		query += " ORDER BY o.date DESC, o.id DESC"

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
			var company, date, typ string
			var startTime, endTime, reason sql.NullString
			var hours float64
			rows.Scan(&id, &company, &date, &typ, &startTime, &endTime, &hours, &reason)
			switch typ {
			case "work":
				sb.WriteString(fmt.Sprintf("  [%d] %s | %s | work | %s–%s | +%.2fh", id, date, company, startTime.String, endTime.String, hours))
			case "reduction":
				sb.WriteString(fmt.Sprintf("  [%d] %s | %s | reduction | -%.2fh", id, date, company, hours))
			}
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

// --- overtime_update ---

type OvertimeUpdateParams struct {
	ID        int     `json:"id" jsonschema:"Overtime entry ID"`
	Date      string  `json:"date,omitempty" jsonschema:"New date (YYYY-MM-DD)"`
	StartTime string  `json:"start_time,omitempty" jsonschema:"New start time (HH:MM); only valid for 'work' entries"`
	EndTime   string  `json:"end_time,omitempty" jsonschema:"New end time (HH:MM); only valid for 'work' entries"`
	Hours     float64 `json:"hours,omitempty" jsonschema:"New hours (only valid for 'reduction' entries; auto-recomputed for 'work')"`
	Reason    string  `json:"reason,omitempty" jsonschema:"New reason"`
}

func makeOvertimeUpdate(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *OvertimeUpdateParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *OvertimeUpdateParams) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}

		var typ string
		var curStart, curEnd sql.NullString
		err := db.QueryRow("SELECT type, start_time, end_time FROM overtime WHERE id = ?", p.ID).Scan(&typ, &curStart, &curEnd)
		if err == sql.ErrNoRows {
			return errResult(fmt.Sprintf("overtime entry with ID %d not found", p.ID))
		}
		if err != nil {
			return errResult(err.Error())
		}

		var sets []string
		var args []any
		if p.Date != "" {
			sets = append(sets, "date = ?")
			args = append(args, p.Date)
		}
		if p.Reason != "" {
			sets = append(sets, "reason = ?")
			args = append(args, p.Reason)
		}

		switch typ {
		case "work":
			newStart := curStart.String
			newEnd := curEnd.String
			if p.StartTime != "" {
				newStart = p.StartTime
			}
			if p.EndTime != "" {
				newEnd = p.EndTime
			}
			if p.StartTime != "" || p.EndTime != "" {
				hours, err := ComputeHours(newStart, newEnd)
				if err != nil {
					return errResult(err.Error())
				}
				sets = append(sets, "start_time = ?", "end_time = ?", "hours = ?")
				args = append(args, newStart, newEnd, hours)
			}
			if p.Hours != 0 {
				return errResult("hours cannot be set directly on a 'work' entry; adjust start_time/end_time")
			}
		case "reduction":
			if p.StartTime != "" || p.EndTime != "" {
				return errResult("start_time/end_time are not allowed on a 'reduction' entry")
			}
			if p.Hours != 0 {
				if p.Hours <= 0 {
					return errResult("hours must be positive")
				}
				sets = append(sets, "hours = ?")
				args = append(args, p.Hours)
			}
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

// --- overtime_delete ---

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

// --- overtime_summary ---

type OvertimeSummaryParams struct {
	Company string `json:"company,omitempty" jsonschema:"Filter by company name or ID"`
	Year    int    `json:"year,omitempty" jsonschema:"Filter by year (e.g. 2026)"`
	Month   int    `json:"month,omitempty" jsonschema:"Filter by month (1-12)"`
}

func makeOvertimeSummary(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *OvertimeSummaryParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *OvertimeSummaryParams) (*mcp.CallToolResult, any, error) {
		query := `SELECT c.name, strftime('%Y-%m', o.date) AS month,
				SUM(CASE WHEN o.type = 'reduction' THEN -o.hours ELSE o.hours END) AS total
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
			sb.WriteString(fmt.Sprintf("  %s | %s | %+.2fh\n", company, month, total))
			grandTotal += total
			count++
		}
		if count == 0 {
			return textResult("No overtime entries found.")
		}
		sb.WriteString(fmt.Sprintf("\n  Total: %+.2fh\n", grandTotal))
		return textResult(sb.String())
	}
}
