# Personal Tracker MCP Server — Design Spec

## Purpose

A personal tracking MCP server for structured time/absence data across multiple employers. Tracks overtime, vacation, and sick days per company with full CRUD and aggregation queries via SQLite.

## Architecture

- New server at `cmd/tracker/main.go` following existing monorepo patterns
- Uses `internal/server.New()` bootstrap, Streamable HTTP on `:8000`
- SQLite via `modernc.org/sqlite` (pure Go, CGO_ENABLED=0 compatible)
- DB path configured via `TRACKER_DB_PATH` env var (default: `/data/tracker.db`)
- Auto-creates tables on startup (no migration tool needed for v1)
- Docker image with `/data` volume for SQLite persistence

## Data Model

### Table: `company`

| Column | Type | Constraints |
|--------|------|-------------|
| id | INTEGER | PRIMARY KEY AUTOINCREMENT |
| name | TEXT | NOT NULL UNIQUE |
| weekly_hours | REAL | |
| annual_vacation_days | INTEGER | |
| created_at | TEXT | NOT NULL DEFAULT (datetime('now')) |

### Table: `overtime`

| Column | Type | Constraints |
|--------|------|-------------|
| id | INTEGER | PRIMARY KEY AUTOINCREMENT |
| company_id | INTEGER | NOT NULL REFERENCES company(id) ON DELETE CASCADE |
| date | TEXT | NOT NULL (YYYY-MM-DD) |
| hours | REAL | NOT NULL |
| reason | TEXT | |
| created_at | TEXT | NOT NULL DEFAULT (datetime('now')) |

Index: `idx_overtime_company_date` on `(company_id, date)`

### Table: `vacation`

| Column | Type | Constraints |
|--------|------|-------------|
| id | INTEGER | PRIMARY KEY AUTOINCREMENT |
| company_id | INTEGER | NOT NULL REFERENCES company(id) ON DELETE CASCADE |
| start_date | TEXT | NOT NULL (YYYY-MM-DD) |
| end_date | TEXT | NOT NULL (YYYY-MM-DD) |
| days | REAL | NOT NULL |
| type | TEXT | NOT NULL DEFAULT 'vacation' |
| note | TEXT | |
| created_at | TEXT | NOT NULL DEFAULT (datetime('now')) |

Index: `idx_vacation_company_year` on `(company_id, start_date)`

Type values: `vacation`, `special_leave`

### Table: `sick_day`

| Column | Type | Constraints |
|--------|------|-------------|
| id | INTEGER | PRIMARY KEY AUTOINCREMENT |
| company_id | INTEGER | NOT NULL REFERENCES company(id) ON DELETE CASCADE |
| start_date | TEXT | NOT NULL (YYYY-MM-DD) |
| end_date | TEXT | NOT NULL (YYYY-MM-DD) |
| days | INTEGER | NOT NULL |
| note | TEXT | |
| created_at | TEXT | NOT NULL DEFAULT (datetime('now')) |

Index: `idx_sick_day_company_year` on `(company_id, start_date)`

## Tools (18 total)

### Company (4)

| Tool | Params | Description |
|------|--------|-------------|
| company_create | name, weekly_hours?, annual_vacation_days? | Create a new company |
| company_list | — | List all companies |
| company_update | id, name?, weekly_hours?, annual_vacation_days? | Update company details |
| company_delete | id | Delete company and all its entries (CASCADE) |

### Overtime (5)

| Tool | Params | Description |
|------|--------|-------------|
| overtime_add | company, date, hours, reason? | Add overtime entry. Company accepts name or ID |
| overtime_list | company?, year?, month? | List overtime entries with optional filters |
| overtime_update | id, date?, hours?, reason? | Update an overtime entry |
| overtime_delete | id | Delete an overtime entry |
| overtime_summary | company?, year?, month? | Sum of overtime hours, grouped by company and/or month |

### Vacation (5)

| Tool | Params | Description |
|------|--------|-------------|
| vacation_add | company, start_date, end_date, days, type?, note? | Add vacation entry |
| vacation_list | company?, year? | List vacation entries with optional filters |
| vacation_update | id, start_date?, end_date?, days?, type?, note? | Update a vacation entry |
| vacation_delete | id | Delete a vacation entry |
| vacation_balance | company?, year? | Remaining vacation days (annual allowance minus taken) |

### Sick Days (4)

| Tool | Params | Description |
|------|--------|-------------|
| sick_day_add | company, start_date, end_date, days, note? | Add sick day entry |
| sick_day_list | company?, year? | List sick day entries with optional filters |
| sick_day_update | id, start_date?, end_date?, days?, note? | Update a sick day entry |
| sick_day_delete | id | Delete a sick day entry |

## Company Resolution

Tools that accept a `company` parameter resolve it as follows:
1. Try to parse as integer → lookup by ID
2. Otherwise → lookup by name (case-insensitive)
3. If not found → return error with list of known companies

## Output Format

All tools return human-readable text (not JSON) for optimal Claude consumption:
- Lists: formatted as tables or line-per-entry
- Summaries: grouped with totals
- Mutations: confirmation message with the affected record

## Files to Create/Modify

### New files
- `cmd/tracker/main.go` — Server entry point, all 18 tools
- `docker/tracker.Dockerfile` — Multi-stage Docker build with `/data` volume

### Modified files
- `go.mod` / `go.sum` — Add `modernc.org/sqlite` dependency
- `Makefile` — Already auto-discovers `cmd/*`, no change needed
- `.github/workflows/build.yml` — Already builds all, no change needed
- `docs/architecture.md` — Add tracker to server list
- `CLAUDE.md` — Add tracker to server overview table

## Docker

```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /bin/server ./cmd/tracker

FROM alpine:3.21
COPY --from=builder /bin/server /server
VOLUME /data
EXPOSE 8000
ENTRYPOINT ["/server"]
```

## Configuration

| Env Variable | Default | Description |
|-------------|---------|-------------|
| TRACKER_DB_PATH | /data/tracker.db | Path to SQLite database file |
| PORT | :8000 | Server listen address |
