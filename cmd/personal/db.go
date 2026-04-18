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
			type TEXT NOT NULL CHECK(type IN ('work','reduction')),
			start_time TEXT,
			end_time TEXT,
			hours REAL NOT NULL CHECK(hours > 0),
			reason TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			CHECK (
				(type='work'      AND start_time IS NOT NULL AND end_time IS NOT NULL) OR
				(type='reduction' AND start_time IS NULL     AND end_time IS NULL)
			)
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

		CREATE TABLE IF NOT EXISTS person (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			note TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS annual_event (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			person_id INTEGER NOT NULL REFERENCES person(id) ON DELETE CASCADE,
			type TEXT NOT NULL CHECK(type IN ('birthday','anniversary','name_day')),
			date TEXT NOT NULL,
			note TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			UNIQUE(person_id, type)
		);
		CREATE INDEX IF NOT EXISTS idx_annual_event_month_day ON annual_event(substr(date, 6));
	`)
	return err
}
