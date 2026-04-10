package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/traffino/mcp-server/internal/config"
	"github.com/traffino/mcp-server/internal/server"

	_ "modernc.org/sqlite"
)

func main() {
	dbPath := config.Get("TRACKER_DB_PATH", "/data/tracker.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatalf("failed to create db directory: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(1)")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	srv := server.New("tracker", "1.0.0")
	s := srv.MCPServer()

	// Company
	mcp.AddTool(s, &mcp.Tool{Name: "company_create", Description: "Create a new company for tracking overtime, vacation, and sick days"}, makeCompanyCreate(db))
	mcp.AddTool(s, &mcp.Tool{Name: "company_list", Description: "List all registered companies"}, makeCompanyList(db))
	mcp.AddTool(s, &mcp.Tool{Name: "company_update", Description: "Update company details (name, weekly hours, vacation days)"}, makeCompanyUpdate(db))
	mcp.AddTool(s, &mcp.Tool{Name: "company_delete", Description: "Delete a company and all its entries"}, makeCompanyDelete(db))

	// Overtime
	mcp.AddTool(s, &mcp.Tool{Name: "overtime_add", Description: "Add an overtime entry for a company"}, makeOvertimeAdd(db))
	mcp.AddTool(s, &mcp.Tool{Name: "overtime_list", Description: "List overtime entries with optional filters"}, makeOvertimeList(db))
	mcp.AddTool(s, &mcp.Tool{Name: "overtime_update", Description: "Update an overtime entry"}, makeOvertimeUpdate(db))
	mcp.AddTool(s, &mcp.Tool{Name: "overtime_delete", Description: "Delete an overtime entry"}, makeOvertimeDelete(db))
	mcp.AddTool(s, &mcp.Tool{Name: "overtime_summary", Description: "Get overtime hours summary, grouped by company and/or month"}, makeOvertimeSummary(db))

	// Vacation
	mcp.AddTool(s, &mcp.Tool{Name: "vacation_add", Description: "Add a vacation entry for a company"}, makeVacationAdd(db))
	mcp.AddTool(s, &mcp.Tool{Name: "vacation_list", Description: "List vacation entries with optional filters"}, makeVacationList(db))
	mcp.AddTool(s, &mcp.Tool{Name: "vacation_update", Description: "Update a vacation entry"}, makeVacationUpdate(db))
	mcp.AddTool(s, &mcp.Tool{Name: "vacation_delete", Description: "Delete a vacation entry"}, makeVacationDelete(db))
	mcp.AddTool(s, &mcp.Tool{Name: "vacation_balance", Description: "Get remaining vacation days (annual allowance minus taken)"}, makeVacationBalance(db))

	// Sick Days
	mcp.AddTool(s, &mcp.Tool{Name: "sick_day_add", Description: "Add a sick day entry for a company"}, makeSickDayAdd(db))
	mcp.AddTool(s, &mcp.Tool{Name: "sick_day_list", Description: "List sick day entries with optional filters"}, makeSickDayList(db))
	mcp.AddTool(s, &mcp.Tool{Name: "sick_day_update", Description: "Update a sick day entry"}, makeSickDayUpdate(db))
	mcp.AddTool(s, &mcp.Tool{Name: "sick_day_delete", Description: "Delete a sick day entry"}, makeSickDayDelete(db))

	srv.ListenAndServe(config.Get("PORT", ":8000"))
}

// --- DB Init ---

func initDB(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS company (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			weekly_hours REAL,
			annual_vacation_days INTEGER,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS overtime (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			company_id INTEGER NOT NULL REFERENCES company(id) ON DELETE CASCADE,
			date TEXT NOT NULL,
			hours REAL NOT NULL,
			reason TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_overtime_company_date ON overtime(company_id, date);

		CREATE TABLE IF NOT EXISTS vacation (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			company_id INTEGER NOT NULL REFERENCES company(id) ON DELETE CASCADE,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			days REAL NOT NULL,
			type TEXT NOT NULL DEFAULT 'vacation',
			note TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_vacation_company_year ON vacation(company_id, start_date);

		CREATE TABLE IF NOT EXISTS sick_day (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			company_id INTEGER NOT NULL REFERENCES company(id) ON DELETE CASCADE,
			start_date TEXT NOT NULL,
			end_date TEXT NOT NULL,
			days INTEGER NOT NULL,
			note TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_sick_day_company_year ON sick_day(company_id, start_date);
	`)
	return err
}

// --- Helpers ---

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

// --- Company Tools ---

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

// --- Overtime Tools ---

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

// --- Vacation Tools ---

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

// --- Sick Day Tools ---

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

// --- Utility ---

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

func currentYear() int {
	return time.Now().Year()
}
