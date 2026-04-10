# Personal Tracker API Coverage

- **API**: Local SQLite database
- **Letzter Check**: 2026-04-10
- **Scope**: full r/w

## Tools

### Company

| Tool | Parameter | Status |
|------|-----------|--------|
| company_create | name, weekly_hours?, annual_vacation_days? | implemented |
| company_list | — | implemented |
| company_update | id, name?, weekly_hours?, annual_vacation_days? | implemented |
| company_delete | id | implemented |

### Overtime

| Tool | Parameter | Status |
|------|-----------|--------|
| overtime_add | company, date, hours, reason? | implemented |
| overtime_list | company?, year?, month? | implemented |
| overtime_update | id, date?, hours?, reason? | implemented |
| overtime_delete | id | implemented |
| overtime_summary | company?, year?, month? | implemented |

### Vacation

| Tool | Parameter | Status |
|------|-----------|--------|
| vacation_add | company, start_date, end_date, days, type?, note? | implemented |
| vacation_list | company?, year? | implemented |
| vacation_update | id, start_date?, end_date?, days?, type?, note? | implemented |
| vacation_delete | id | implemented |
| vacation_balance | company?, year? | implemented |

### Sick Days

| Tool | Parameter | Status |
|------|-----------|--------|
| sick_day_add | company, start_date, end_date, days, note? | implemented |
| sick_day_list | company?, year? | implemented |
| sick_day_update | id, start_date?, end_date?, days?, note? | implemented |
| sick_day_delete | id | implemented |

## Hinweise

- SQLite via `modernc.org/sqlite` (pure Go, kein CGO)
- DB-Pfad konfigurierbar ueber `TRACKER_DB_PATH` (default: `/data/tracker.db`)
- Company-Parameter akzeptiert Name (case-insensitive) oder ID
- Vacation type: `vacation` (default) oder `special_leave`
- Overtime hours koennen negativ sein (Abbau)
