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

		CREATE TABLE IF NOT EXISTS project (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			company_id INTEGER REFERENCES company(id) ON DELETE SET NULL,
			status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','archived')),
			note TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			UNIQUE(company_id, name)
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_project_private_name
			ON project(name) WHERE company_id IS NULL;
		CREATE INDEX IF NOT EXISTS idx_project_company_status
			ON project(company_id, status);

		CREATE TABLE IF NOT EXISTS todo (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			project_id INTEGER REFERENCES project(id) ON DELETE SET NULL,
			company_id INTEGER REFERENCES company(id) ON DELETE SET NULL,
			parent_id INTEGER REFERENCES todo(id) ON DELETE CASCADE,
			status TEXT NOT NULL DEFAULT 'open'
				CHECK(status IN ('open','in_progress','waiting','done','cancelled')),
			due_date TEXT,
			note TEXT,
			recurrence_pattern TEXT,
			completed_at TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_todo_status_due ON todo(status, due_date);
		CREATE INDEX IF NOT EXISTS idx_todo_project ON todo(project_id);
		CREATE INDEX IF NOT EXISTS idx_todo_parent ON todo(parent_id);

		CREATE TABLE IF NOT EXISTS todo_completion (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			todo_id INTEGER NOT NULL REFERENCES todo(id) ON DELETE CASCADE,
			completed_at TEXT NOT NULL,
			due_date_at_completion TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE INDEX IF NOT EXISTS idx_todo_completion_todo ON todo_completion(todo_id, completed_at);
	`)
	return err
}
