package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/traffino/mcp-server/internal/config"
	"github.com/traffino/mcp-server/internal/server"

	_ "modernc.org/sqlite"
)

func main() {
	if err := initTimezone(config.Get("PERSONAL_TZ", "Europe/Berlin")); err != nil {
		log.Fatalf("failed to init timezone: %v", err)
	}

	dbPath := config.Get("PERSONAL_DB_PATH", "/data/personal.db")
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

	srv := server.New("personal", "1.0.0")
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
