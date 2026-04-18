package main

import "database/sql"

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
