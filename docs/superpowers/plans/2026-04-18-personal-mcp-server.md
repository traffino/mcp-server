# Personal MCP Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Tracker-MCP-Server in `personal` umbenennen und zu Productivity-Hub ausbauen: Overtime mit Zeiten + Reduction-Typ, Person/Annual Event, Projekte, dreistufige TODOs mit Rekurrenz.

**Architecture:** Ein Binary `cmd/personal/` mit mehreren Dateien in `package main`, aufgesplittet nach Domain. Persistenz via SQLite (`modernc.org/sqlite`), Konfig via Env-Vars (`PERSONAL_DB_PATH`, `PERSONAL_TZ`). Reine Funktionen (Recurrence, Time-Helpers) mit TDD; Tool-Bodies folgen dem bestehenden Muster aus `cmd/tracker/main.go`.

**Tech Stack:** Go 1.26, `modernc.org/sqlite`, `github.com/modelcontextprotocol/go-sdk`.

**Working Mode:** Direkt auf `main` des `mcp-server`-Repos (kein Worktree — User-Preference). Fünf Commits im `mcp-server`-Repo plus ein Commit in `infrastructure-home`. Nach jedem Commit: push zu origin.

**Referenzen:**
- Spec: `docs/superpowers/specs/2026-04-18-personal-mcp-server-design.md`
- Vorlagen-Code: `cmd/tracker/main.go` (wird zerlegt + umbenannt)

---

## File Structure

### Endzustand `cmd/personal/` (nach Plan-Abschluss)

| Datei              | Verantwortung                                                       |
|--------------------|---------------------------------------------------------------------|
| `main.go`          | Bootstrap, Env-Config, Tool-Registration, Timezone-Init, Shutdown   |
| `db.go`            | Schema-DDL, Migrationen via `PRAGMA user_version`                   |
| `helpers.go`       | `errResult`/`textResult`/`nilIf*`, Resolver (`resolveCompany`, ...) |
| `time.go` + Test   | Datum/Uhrzeit-Parser, `ComputeHours`, `Today`, `NowString`          |
| `recurrence.go` + Test | `Pattern` Struct, `ParsePattern`, `NextOccurrence`              |
| `company.go`       | 4 Company Tools                                                     |
| `overtime.go`      | 6 Overtime Tools (work + reduction)                                 |
| `vacation.go`      | 5 Vacation Tools                                                    |
| `sickday.go`       | 4 Sick Day Tools                                                    |
| `person.go`        | 4 Person Tools                                                      |
| `annual_event.go`  | 5 Annual Event Tools                                                |
| `project.go`       | 5 Project Tools                                                     |
| `todo.go`          | 9 TODO/Subtask Tools                                                |

### Weitere Änderungen

| Pfad                                     | Änderung                              |
|------------------------------------------|---------------------------------------|
| `docker/tracker.Dockerfile`              | → `docker/personal.Dockerfile`        |
| `README.md`                              | Server-Tabelle + Coverage-Tabelle     |
| `CLAUDE.md`                              | Server-Übersicht                      |
| `docs/api-coverage/tracker.md`           | → `docs/api-coverage/personal.md`     |
| `infrastructure-home` (separater Repo)   | Docker-Compose-Service umbenennen     |

---

## Phase 1 — Rename + Datei-Split (Commit 1)

Ziel: `cmd/tracker/` ersatzlos ablösen durch `cmd/personal/` mit identischer Funktionalität, aber in mehreren Dateien. Neu: `PERSONAL_DB_PATH`/`PERSONAL_TZ` Env-Vars. Kein Tool-Verhalten ändert sich.

### Task 1: Neues Verzeichnis anlegen und Platzhalter committen

**Files:**
- Create: `cmd/personal/.gitkeep`

- [ ] **Step 1: Leeres Verzeichnis erstellen**

```bash
cd ~/Projects/private/traffino/mcp-server
mkdir -p cmd/personal
touch cmd/personal/.gitkeep
```

- [ ] **Step 2: Vorhandenes Repo-Setup prüfen (Pull, aktueller Stand)**

```bash
git status
git pull --ff-only
```
Expected: `working tree clean`, `Already up to date`.

---

### Task 2: `helpers.go` aus `cmd/tracker/main.go` extrahieren

**Files:**
- Create: `cmd/personal/helpers.go`
- Reference: `cmd/tracker/main.go:119-168, 877-900`

- [ ] **Step 1: `helpers.go` anlegen**

```go
package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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
```

---

### Task 3: `time.go` mit Timezone-State + `currentYear`

**Files:**
- Create: `cmd/personal/time.go`

- [ ] **Step 1: `time.go` anlegen**

```go
package main

import (
	"fmt"
	"time"
)

// appLoc holds the application timezone, set at bootstrap via PERSONAL_TZ.
var appLoc = time.UTC

func initTimezone(tzName string) error {
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return fmt.Errorf("invalid timezone %q: %w", tzName, err)
	}
	appLoc = loc
	return nil
}

func today() string {
	return time.Now().In(appLoc).Format("2006-01-02")
}

func nowString() string {
	return time.Now().In(appLoc).Format("2006-01-02 15:04:05")
}

func currentYear() int {
	return time.Now().In(appLoc).Year()
}
```

*Die weiteren Helper (`ParseTime`, `ComputeHours`) kommen erst in Phase 2 mit TDD. Hier nur, was `cmd/tracker/main.go` schon nutzt: `currentYear`.*

---

### Task 4: `db.go` mit initDB (unveränderter SQL-Block aus tracker)

**Files:**
- Create: `cmd/personal/db.go`
- Reference: `cmd/tracker/main.go:71-115`

- [ ] **Step 1: `db.go` anlegen**

```go
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
```

*Bewusst noch OHNE `PRAGMA user_version`-Versionierung und ohne die neuen Tabellen — das kommt in späteren Phasen als explizite Schema-Evolution. In dieser Phase ist Verhalten 1:1 mit tracker.*

---

### Task 5: `company.go` (Company-Tools 1:1 aus tracker)

**Files:**
- Create: `cmd/personal/company.go`
- Reference: `cmd/tracker/main.go:170-295`

- [ ] **Step 1: `company.go` anlegen**

Kopiere den Block `// --- Company Tools ---` bis vor `// --- Overtime Tools ---` aus `cmd/tracker/main.go` wortwörtlich. Mit Package-Header:

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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

// ... rest of Company Tools copied 1:1 from cmd/tracker/main.go lines 196-295
// (makeCompanyList, CompanyUpdateParams + makeCompanyUpdate, CompanyDeleteParams + makeCompanyDelete)
```

*Instruction: Copy `makeCompanyList`, `CompanyUpdateParams`, `makeCompanyUpdate`, `CompanyDeleteParams`, `makeCompanyDelete` verbatim from lines 196-295 of the original file.*

---

### Task 6: `overtime.go` (Overtime-Tools 1:1 aus tracker, noch alte Variante)

**Files:**
- Create: `cmd/personal/overtime.go`
- Reference: `cmd/tracker/main.go:297-508`

- [ ] **Step 1: `overtime.go` anlegen**

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ... copy the entire "// --- Overtime Tools ---" section from lines 297-508 verbatim
// (OvertimeAddParams, makeOvertimeAdd, OvertimeListParams, makeOvertimeList,
//  OvertimeUpdateParams, makeOvertimeUpdate, IDParam, makeOvertimeDelete,
//  OvertimeSummaryParams, makeOvertimeSummary)
```

**Wichtig**: `IDParam` ist in den Originalquellen an dieser Stelle definiert. In der neuen Aufteilung behält `overtime.go` die Definition, weil es der erste Nutzer ist. `vacation.go` und `sickday.go` importieren es als Paket-internen Typ mit (benutzen den gleichen Typnamen).

---

### Task 7: `vacation.go` (Vacation-Tools 1:1 aus tracker)

**Files:**
- Create: `cmd/personal/vacation.go`
- Reference: `cmd/tracker/main.go:510-726`

- [ ] **Step 1: `vacation.go` anlegen**

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Copy "// --- Vacation Tools ---" block (VacationAddParams, makeVacationAdd,
// VacationListParams, makeVacationList, VacationUpdateParams, makeVacationUpdate,
// makeVacationDelete, VacationBalanceParams, makeVacationBalance) from lines 510-726.
```

*Hinweis: `makeVacationDelete` nutzt `IDParam`, das aus `overtime.go` sichtbar ist (gleiches Package).*

---

### Task 8: `sickday.go` (Sick-Day-Tools 1:1 aus tracker)

**Files:**
- Create: `cmd/personal/sickday.go`
- Reference: `cmd/tracker/main.go:728-873`

- [ ] **Step 1: `sickday.go` anlegen**

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Copy "// --- Sick Day Tools ---" block (SickDayAddParams, makeSickDayAdd,
// SickDayListParams, makeSickDayList, SickDayUpdateParams, makeSickDayUpdate,
// makeSickDayDelete) from lines 728-873.
```

---

### Task 9: `main.go` neu schreiben mit Bootstrap + Timezone-Init

**Files:**
- Create: `cmd/personal/main.go`

- [ ] **Step 1: `main.go` anlegen**

```go
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
```

- [ ] **Step 2: Platzhalter `.gitkeep` entfernen**

```bash
rm cmd/personal/.gitkeep
```

---

### Task 10: `cmd/tracker/` löschen + Dockerfile umbenennen

**Files:**
- Delete: `cmd/tracker/main.go`, `cmd/tracker/` directory
- Rename: `docker/tracker.Dockerfile` → `docker/personal.Dockerfile`

- [ ] **Step 1: Alten tracker-Ordner löschen**

```bash
cd ~/Projects/private/traffino/mcp-server
git rm -rf cmd/tracker
```

- [ ] **Step 2: Dockerfile umbenennen**

```bash
git mv docker/tracker.Dockerfile docker/personal.Dockerfile
```

- [ ] **Step 3: Dockerfile-Inhalt anpassen**

```go
// docker/personal.Dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /bin/server ./cmd/personal

FROM alpine:3.21
COPY --from=builder /bin/server /server
VOLUME /data
EXPOSE 8000
ENTRYPOINT ["/server"]
```

---

### Task 11: API-Coverage-Doc umbenennen

**Files:**
- Rename: `docs/api-coverage/tracker.md` → `docs/api-coverage/personal.md`

- [ ] **Step 1: Datei umbenennen**

```bash
git mv docs/api-coverage/tracker.md docs/api-coverage/personal.md
```

- [ ] **Step 2: Kopfzeilen anpassen** (in `docs/api-coverage/personal.md`)

```markdown
# Personal MCP Server API Coverage

- **API**: Local SQLite database
- **Letzter Check**: 2026-04-18
- **Scope**: full r/w
```

- [ ] **Step 3: Hinweise am Ende aktualisieren**

Ersetze den Block `## Hinweise` durch:

```markdown
## Hinweise

- SQLite via `modernc.org/sqlite` (pure Go, kein CGO)
- DB-Pfad konfigurierbar ueber `PERSONAL_DB_PATH` (default: `/data/personal.db`)
- Timezone ueber `PERSONAL_TZ` (IANA, default: `Europe/Berlin`)
- Company-Parameter akzeptiert Name (case-insensitive) oder ID
- Vacation type: `vacation` (default) oder `special_leave`
- Overtime hours koennen negativ sein (Abbau)

*Hinweis: Phase-1-Zustand. Nach Abschluss aller Phasen wird diese Datei mit Person/Annual Event/Project/Todo-Sektionen und angepasstem Overtime-Schema erweitert.*
```

---

### Task 12: README.md anpassen

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Zeile in Server-Tabelle (Zeilen 11-22) — `tracker` entfernen und `personal` hinzufügen**

Suche:
```
| aggregator | MCP Proxy | Tool aggregation across backends | done |
```

Ersetze den Block oberhalb mit gleicher Zeile und füge `personal` ein:
```
| bunq | Bunq Banking | Accounts, Payments, Cards, Schedules (read-only) | done |
| personal | Local SQLite | Companies, Overtime, Vacation, Sick Days, People, Events, Projects, TODOs | done |
| aggregator | MCP Proxy | Tool aggregation across backends | done |
```

- [ ] **Step 2: Coverage-Tabelle (Zeilen 71-77) — Eintrag `tracker` → `personal`**

Ersetze die Coverage-Tabelle am Ende der Datei mit einer erweiterten Fassung, die `personal` listet:
```markdown
| Server | Coverage Doc | Endpoints |
|--------|-------------|-----------|
| brave | [brave.md](docs/api-coverage/brave.md) | Web Search, Suggest |
| bunq | [bunq.md](docs/api-coverage/bunq.md) | Accounts, Payments, Cards, Schedules |
| discord | [discord.md](docs/api-coverage/discord.md) | Guilds, Channels, Roles, Reactions, Threads, Users |
| docker | [docker.md](docs/api-coverage/docker.md) | Containers, Images, Networks, Volumes, System |
| github | [github.md](docs/api-coverage/github.md) | Repos, Issues, PRs, Actions, Releases, Search, Users |
| hetzner | [hetzner.md](docs/api-coverage/hetzner.md) | Servers, SSH Keys, Firewalls, Networks, Volumes, IPs, LBs, ... |
| ms365 | [ms365.md](docs/api-coverage/ms365.md) | Mail, Calendar, Contacts, OneDrive, Teams, OneNote, To Do |
| personal | [personal.md](docs/api-coverage/personal.md) | Companies, Overtime, Vacation, Sick Days, People, Annual Events, Projects, TODOs |
```

---

### Task 13: CLAUDE.md anpassen

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: Server-Übersicht (Zeile 44-59) — tracker-Zeile ersetzen**

Suche:
```
| tracker | Personal Tracker | Ueberstunden, Urlaub, Kranktage (SQLite) | MCP Server |
```

Ersetze mit:
```
| personal | Personal Productivity | Ueberstunden, Urlaub, Kranktage, People, Events, Projekte, TODOs (SQLite) | MCP Server |
```

---

### Task 14: Build verifizieren + Commit

- [ ] **Step 1: Build-Ziele prüfen (`make` erkennt neue Binaries via Wildcard)**

```bash
cd ~/Projects/private/traffino/mcp-server
make personal
```
Expected: Binary in `build/personal`, keine Errors.

- [ ] **Step 2: Alle Server bauen (Monorepo-Vollbuild)**

```bash
make
```
Expected: Alle Binaries kompilieren grün.

- [ ] **Step 3: Go-Tests ausführen (sollten leer aber fehlerfrei laufen)**

```bash
make test
```
Expected: `ok  	github.com/traffino/mcp-server/...` — keine Testsuites in Phase 1.

- [ ] **Step 4: Git-Status prüfen**

```bash
git status
```
Expected: Gelöschte Dateien in `cmd/tracker/`, `docker/tracker.Dockerfile`, `docs/api-coverage/tracker.md`; neue in `cmd/personal/`, `docker/personal.Dockerfile`, `docs/api-coverage/personal.md`; modifizierte `README.md`, `CLAUDE.md`.

- [ ] **Step 5: Commit + Push**

```bash
git add -A
git commit -m "$(cat <<'EOF'
refactor(personal): rename tracker to personal and split into domain files

- Move cmd/tracker/ to cmd/personal/, split monolithic main.go into
  main.go, db.go, helpers.go, time.go, company.go, overtime.go,
  vacation.go, sickday.go.
- Rename docker/tracker.Dockerfile, docs/api-coverage/tracker.md.
- Add PERSONAL_TZ env var (default Europe/Berlin) and PERSONAL_DB_PATH
  (default /data/personal.db). TRACKER_DB_PATH no longer used.
- Update README.md and CLAUDE.md server tables.

Tool surface unchanged in this commit; behavior identical to tracker.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
git push
```

---

## Phase 2 — Overtime v2 mit Zeiten und Reduction-Typ (Commit 2)

Ziel: Overtime-Schema erweitern um `type`/`start_time`/`end_time`, `overtime_add` aufteilen in `overtime_add_work` + `overtime_add_reduction`, Summary-Query auf `CASE`-Diskriminierung umstellen. Neue `time.go`-Helper mit TDD.

### Task 15: `time_test.go` — `ComputeHours` Failing Tests schreiben

**Files:**
- Create: `cmd/personal/time_test.go`

- [ ] **Step 1: Test-Datei anlegen**

```go
package main

import (
	"math"
	"testing"
)

func TestComputeHours(t *testing.T) {
	tests := []struct {
		name      string
		start     string
		end       string
		want      float64
		wantError bool
	}{
		{"full hour", "09:00", "17:00", 8.0, false},
		{"half hour", "09:00", "09:30", 0.5, false},
		{"quarter", "08:00", "08:15", 0.25, false},
		{"with minutes", "08:30", "10:15", 1.75, false},
		{"end before start", "10:00", "09:00", 0, true},
		{"equal", "10:00", "10:00", 0, true},
		{"bad format start", "9-00", "10:00", 0, true},
		{"bad format end", "09:00", "10:0Z", 0, true},
		{"minute precision 2dp", "08:00", "08:20", 0.33, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ComputeHours(tc.start, tc.end)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error, got hours=%v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if math.Abs(got-tc.want) > 0.001 {
				t.Errorf("ComputeHours(%q, %q) = %v, want %v", tc.start, tc.end, got, tc.want)
			}
		})
	}
}

func TestParseTimeOfDay(t *testing.T) {
	if _, err := ParseTimeOfDay("09:30"); err != nil {
		t.Errorf("ParseTimeOfDay(09:30) failed: %v", err)
	}
	if _, err := ParseTimeOfDay("25:00"); err == nil {
		t.Errorf("ParseTimeOfDay(25:00) should fail")
	}
	if _, err := ParseTimeOfDay("9:00"); err == nil {
		t.Errorf("ParseTimeOfDay(9:00) without leading zero should fail")
	}
}
```

- [ ] **Step 2: Tests laufen lassen — rot**

```bash
cd ~/Projects/private/traffino/mcp-server
go test ./cmd/personal/ -run 'TestComputeHours|TestParseTimeOfDay' -v
```
Expected: FAIL mit `undefined: ComputeHours` / `undefined: ParseTimeOfDay`.

---

### Task 16: `time.go` — `ComputeHours` + `ParseTimeOfDay` implementieren

**Files:**
- Modify: `cmd/personal/time.go`

- [ ] **Step 1: Funktionen ergänzen**

Am Ende von `cmd/personal/time.go`:

```go
// ParseTimeOfDay parses "HH:MM" strictly (leading zeros required, 00..23 / 00..59).
func ParseTimeOfDay(s string) (time.Time, error) {
	return time.Parse("15:04", s)
}

// ComputeHours returns (end - start) in hours, rounded to 2 decimal places.
// Both inputs must be "HH:MM". end must be strictly after start.
func ComputeHours(start, end string) (float64, error) {
	s, err := ParseTimeOfDay(start)
	if err != nil {
		return 0, fmt.Errorf("start_time must be HH:MM: %w", err)
	}
	e, err := ParseTimeOfDay(end)
	if err != nil {
		return 0, fmt.Errorf("end_time must be HH:MM: %w", err)
	}
	if !e.After(s) {
		return 0, fmt.Errorf("end_time must be after start_time")
	}
	diff := e.Sub(s).Minutes()
	hours := diff / 60.0
	return float64(int(hours*100+0.5)) / 100.0, nil
}
```

- [ ] **Step 2: Tests laufen lassen — grün**

```bash
go test ./cmd/personal/ -run 'TestComputeHours|TestParseTimeOfDay' -v
```
Expected: PASS für alle Cases.

---

### Task 17: Schema-Evolution mit `PRAGMA user_version` einführen

**Files:**
- Modify: `cmd/personal/db.go`

- [ ] **Step 1: `initDB` refaktorieren — Migrationen-Sequenz**

Ersetze den Inhalt von `cmd/personal/db.go` komplett durch:

```go
package main

import (
	"database/sql"
	"fmt"
)

func initDB(db *sql.DB) error {
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}

	migrations := []func(*sql.DB) error{
		migrateToV1, // target user_version 1
		migrateToV2, // target user_version 2
	}

	for i := version; i < len(migrations); i++ {
		if err := migrations[i](db); err != nil {
			return fmt.Errorf("migration to v%d: %w", i+1, err)
		}
		if _, err := db.Exec(fmt.Sprintf("PRAGMA user_version = %d", i+1)); err != nil {
			return fmt.Errorf("bump user_version to %d: %w", i+1, err)
		}
	}
	return nil
}

func migrateToV1(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS company (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			weekly_hours REAL,
			annual_vacation_days INTEGER,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

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

func migrateToV2(db *sql.DB) error {
	_, err := db.Exec(`
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
	`)
	return err
}
```

*Hinweis: `overtime` ist bewusst in V2 gewandert. Da keine Migration bestehender Daten nötig ist (Fresh-Start), wird `overtime` erst mit dem neuen Schema angelegt.*

---

### Task 18: `overtime.go` komplett neu schreiben (work + reduction)

**Files:**
- Modify: `cmd/personal/overtime.go` (kompletter Rewrite)

- [ ] **Step 1: Datei durch Neufassung ersetzen**

```go
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
```

---

### Task 19: `main.go` — Tool-Registration für Overtime anpassen

**Files:**
- Modify: `cmd/personal/main.go`

- [ ] **Step 1: Overtime-Registration ersetzen**

Suche:
```go
	// Overtime
	mcp.AddTool(s, &mcp.Tool{Name: "overtime_add", Description: "Add an overtime entry for a company"}, makeOvertimeAdd(db))
	mcp.AddTool(s, &mcp.Tool{Name: "overtime_list", Description: "List overtime entries with optional filters"}, makeOvertimeList(db))
	mcp.AddTool(s, &mcp.Tool{Name: "overtime_update", Description: "Update an overtime entry"}, makeOvertimeUpdate(db))
	mcp.AddTool(s, &mcp.Tool{Name: "overtime_delete", Description: "Delete an overtime entry"}, makeOvertimeDelete(db))
	mcp.AddTool(s, &mcp.Tool{Name: "overtime_summary", Description: "Get overtime hours summary, grouped by company and/or month"}, makeOvertimeSummary(db))
```

Ersetze durch:
```go
	// Overtime
	mcp.AddTool(s, &mcp.Tool{Name: "overtime_add_work", Description: "Add a worked overtime entry (with start/end time). Hours computed from times."}, makeOvertimeAddWork(db))
	mcp.AddTool(s, &mcp.Tool{Name: "overtime_add_reduction", Description: "Add an overtime reduction (compensatory time off). Hours subtracted from balance."}, makeOvertimeAddReduction(db))
	mcp.AddTool(s, &mcp.Tool{Name: "overtime_list", Description: "List overtime entries with optional filters (company, year, month, type)"}, makeOvertimeList(db))
	mcp.AddTool(s, &mcp.Tool{Name: "overtime_update", Description: "Update an overtime entry. For 'work' entries: change times auto-recomputes hours. For 'reduction': set hours directly."}, makeOvertimeUpdate(db))
	mcp.AddTool(s, &mcp.Tool{Name: "overtime_delete", Description: "Delete an overtime entry"}, makeOvertimeDelete(db))
	mcp.AddTool(s, &mcp.Tool{Name: "overtime_summary", Description: "Get overtime hours summary (net balance), grouped by company and month"}, makeOvertimeSummary(db))
```

---

### Task 20: API-Coverage-Doc aktualisieren (Overtime-Sektion)

**Files:**
- Modify: `docs/api-coverage/personal.md`

- [ ] **Step 1: Overtime-Tabelle ersetzen**

Ersetze den `### Overtime` Abschnitt durch:

```markdown
### Overtime

| Tool | Parameter | Status |
|------|-----------|--------|
| overtime_add_work | company, date, start_time, end_time, reason? | implemented |
| overtime_add_reduction | company, date, hours, reason? | implemented |
| overtime_list | company?, year?, month?, type? | implemented |
| overtime_update | id, date?, start_time?, end_time?, hours?, reason? | implemented |
| overtime_delete | id | implemented |
| overtime_summary | company?, year?, month? | implemented |
```

- [ ] **Step 2: Hinweise anpassen** — Zeile `Overtime hours koennen negativ sein (Abbau)` ersetzen durch:

```markdown
- Overtime hat zwei Typen: `work` (mit start_time/end_time, hours berechnet) und `reduction` (Abbau, hours positiv gespeichert, im Saldo negativ)
```

---

### Task 21: Build + Tests + Commit

- [ ] **Step 1: Build**

```bash
cd ~/Projects/private/traffino/mcp-server
make personal
```
Expected: Grün.

- [ ] **Step 2: Tests**

```bash
make test
```
Expected: `TestComputeHours` und `TestParseTimeOfDay` passen grün.

- [ ] **Step 3: Commit + Push**

```bash
git add -A
git commit -m "$(cat <<'EOF'
feat(personal): overtime with start/end times and reduction type

- Extend overtime schema with type ('work'|'reduction'), start_time,
  end_time columns and CHECK constraints; hours stored positive.
- Split overtime_add into overtime_add_work (times required, hours
  computed) and overtime_add_reduction (hours positive, summed
  negatively in balance).
- overtime_list gains type filter; overtime_update handles both types.
- overtime_summary uses CASE WHEN type='reduction' THEN -hours ELSE hours
  for net balance.
- Add ComputeHours and ParseTimeOfDay in time.go with tests.
- Add PRAGMA user_version migration scaffolding in db.go.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
git push
```

---

## Phase 3 — Person + Annual Event (Commit 3)

Ziel: Zwei neue Domänen. `person` als schlanke Tabelle (ID/Name/Note), `annual_event` für Geburtstage/Jahrestage/Namenstage mit Monatsfilter + Upcoming-Query.

### Task 22: Schema-Migration V3 hinzufügen

**Files:**
- Modify: `cmd/personal/db.go`

- [ ] **Step 1: `migrateToV3` in `migrations`-Slice einfügen und Funktion anlegen**

Am Slice-Ende:
```go
	migrations := []func(*sql.DB) error{
		migrateToV1,
		migrateToV2,
		migrateToV3,
	}
```

Funktion am Dateiende ergänzen:

```go
func migrateToV3(db *sql.DB) error {
	_, err := db.Exec(`
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
```

---

### Task 23: `resolvePerson` in `helpers.go` ergänzen

**Files:**
- Modify: `cmd/personal/helpers.go`

- [ ] **Step 1: Funktion am Dateiende anfügen**

```go
// resolvePerson resolves a person by name (case-insensitive) or ID.
func resolvePerson(db *sql.DB, input string) (int64, string, error) {
	if id, err := strconv.ParseInt(input, 10, 64); err == nil {
		var name string
		err := db.QueryRow("SELECT name FROM person WHERE id = ?", id).Scan(&name)
		if err == sql.ErrNoRows {
			return 0, "", fmt.Errorf("person with ID %d not found", id)
		}
		if err != nil {
			return 0, "", err
		}
		return id, name, nil
	}
	var id int64
	var name string
	err := db.QueryRow("SELECT id, name FROM person WHERE lower(name) = lower(?)", input).Scan(&id, &name)
	if err == sql.ErrNoRows {
		return 0, "", fmt.Errorf("person %q not found", input)
	}
	if err != nil {
		return 0, "", err
	}
	return id, name, nil
}
```

---

### Task 24: `person.go` — 4 Tools

**Files:**
- Create: `cmd/personal/person.go`

- [ ] **Step 1: Datei anlegen**

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type PersonAddParams struct {
	Name string `json:"name" jsonschema:"Person name (unique)"`
	Note string `json:"note,omitempty" jsonschema:"Optional note"`
}

func makePersonAdd(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *PersonAddParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *PersonAddParams) (*mcp.CallToolResult, any, error) {
		if p.Name == "" {
			return errResult("name is required")
		}
		res, err := db.Exec("INSERT INTO person (name, note) VALUES (?, ?)", p.Name, nilIfEmpty(p.Note))
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return errResult(fmt.Sprintf("person %q already exists", p.Name))
			}
			return errResult(err.Error())
		}
		id, _ := res.LastInsertId()
		return textResult(fmt.Sprintf("Added person %q (ID: %d)", p.Name, id))
	}
}

type PersonListParams struct {
	Search string `json:"search,omitempty" jsonschema:"Case-insensitive name substring filter"`
}

func makePersonList(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *PersonListParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *PersonListParams) (*mcp.CallToolResult, any, error) {
		query := "SELECT id, name, note FROM person WHERE 1=1"
		var args []any
		if p.Search != "" {
			query += " AND lower(name) LIKE ?"
			args = append(args, "%"+strings.ToLower(p.Search)+"%")
		}
		query += " ORDER BY name"
		rows, err := db.Query(query, args...)
		if err != nil {
			return errResult(err.Error())
		}
		defer rows.Close()

		var sb strings.Builder
		sb.WriteString("People:\n\n")
		count := 0
		for rows.Next() {
			var id int64
			var name string
			var note sql.NullString
			rows.Scan(&id, &name, &note)
			sb.WriteString(fmt.Sprintf("  [%d] %s", id, name))
			if note.Valid {
				sb.WriteString(fmt.Sprintf(" — %s", note.String))
			}
			sb.WriteString("\n")
			count++
		}
		if count == 0 {
			return textResult("No people found.")
		}
		return textResult(sb.String())
	}
}

type PersonUpdateParams struct {
	ID   int    `json:"id" jsonschema:"Person ID"`
	Name string `json:"name,omitempty" jsonschema:"New name"`
	Note string `json:"note,omitempty" jsonschema:"New note"`
}

func makePersonUpdate(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *PersonUpdateParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *PersonUpdateParams) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		var sets []string
		var args []any
		if p.Name != "" {
			sets = append(sets, "name = ?")
			args = append(args, p.Name)
		}
		if p.Note != "" {
			sets = append(sets, "note = ?")
			args = append(args, p.Note)
		}
		if len(sets) == 0 {
			return errResult("nothing to update")
		}
		args = append(args, p.ID)
		res, err := db.Exec("UPDATE person SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
		if err != nil {
			return errResult(err.Error())
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return errResult(fmt.Sprintf("person with ID %d not found", p.ID))
		}
		return textResult(fmt.Sprintf("Updated person ID %d", p.ID))
	}
}

func makePersonDelete(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *IDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IDParam) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		var name string
		err := db.QueryRow("SELECT name FROM person WHERE id = ?", p.ID).Scan(&name)
		if err == sql.ErrNoRows {
			return errResult(fmt.Sprintf("person with ID %d not found", p.ID))
		}
		if err != nil {
			return errResult(err.Error())
		}
		_, err = db.Exec("DELETE FROM person WHERE id = ?", p.ID)
		if err != nil {
			return errResult(err.Error())
		}
		return textResult(fmt.Sprintf("Deleted person %q (ID: %d) and all their events", name, p.ID))
	}
}
```

---

### Task 25: `annual_event.go` — 5 Tools

**Files:**
- Create: `cmd/personal/annual_event.go`

- [ ] **Step 1: Datei anlegen**

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var allowedEventTypes = map[string]bool{
	"birthday":    true,
	"anniversary": true,
	"name_day":    true,
}

type AnnualEventAddParams struct {
	Person string `json:"person" jsonschema:"Person name or ID"`
	Type   string `json:"type" jsonschema:"Event type: birthday, anniversary, or name_day"`
	Date   string `json:"date" jsonschema:"Date of first occurrence (YYYY-MM-DD); year used for age/anniversary number"`
	Note   string `json:"note,omitempty" jsonschema:"Optional note"`
}

func makeAnnualEventAdd(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *AnnualEventAddParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *AnnualEventAddParams) (*mcp.CallToolResult, any, error) {
		if p.Person == "" || p.Type == "" || p.Date == "" {
			return errResult("person, type, and date are required")
		}
		if !allowedEventTypes[p.Type] {
			return errResult("type must be one of: birthday, anniversary, name_day")
		}
		if _, err := time.Parse("2006-01-02", p.Date); err != nil {
			return errResult("date must be YYYY-MM-DD")
		}
		personID, personName, err := resolvePerson(db, p.Person)
		if err != nil {
			return errResult(err.Error())
		}
		res, err := db.Exec(
			"INSERT INTO annual_event (person_id, type, date, note) VALUES (?, ?, ?, ?)",
			personID, p.Type, p.Date, nilIfEmpty(p.Note),
		)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return errResult(fmt.Sprintf("%s for %s already exists", p.Type, personName))
			}
			return errResult(err.Error())
		}
		id, _ := res.LastInsertId()
		return textResult(fmt.Sprintf("Added %s for %s on %s (ID: %d)", p.Type, personName, p.Date, id))
	}
}

type AnnualEventListParams struct {
	Person string `json:"person,omitempty" jsonschema:"Filter by person name or ID"`
	Type   string `json:"type,omitempty" jsonschema:"Filter by type"`
	Month  int    `json:"month,omitempty" jsonschema:"Filter by month (1-12)"`
}

func makeAnnualEventList(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *AnnualEventListParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *AnnualEventListParams) (*mcp.CallToolResult, any, error) {
		query := `SELECT e.id, p.name, e.type, e.date, e.note
			FROM annual_event e JOIN person p ON e.person_id = p.id WHERE 1=1`
		var args []any
		if p.Person != "" {
			personID, _, err := resolvePerson(db, p.Person)
			if err != nil {
				return errResult(err.Error())
			}
			query += " AND e.person_id = ?"
			args = append(args, personID)
		}
		if p.Type != "" {
			if !allowedEventTypes[p.Type] {
				return errResult("type must be one of: birthday, anniversary, name_day")
			}
			query += " AND e.type = ?"
			args = append(args, p.Type)
		}
		if p.Month > 0 {
			if p.Month > 12 {
				return errResult("month must be 1-12")
			}
			query += " AND substr(e.date, 6, 2) = ?"
			args = append(args, fmt.Sprintf("%02d", p.Month))
		}
		query += " ORDER BY substr(e.date, 6), p.name"

		rows, err := db.Query(query, args...)
		if err != nil {
			return errResult(err.Error())
		}
		defer rows.Close()

		var sb strings.Builder
		sb.WriteString("Annual events:\n\n")
		count := 0
		for rows.Next() {
			var id int64
			var person, typ, date string
			var note sql.NullString
			rows.Scan(&id, &person, &typ, &date, &note)
			sb.WriteString(fmt.Sprintf("  [%d] %s | %s | %s", id, date, person, typ))
			if note.Valid {
				sb.WriteString(fmt.Sprintf(" | %s", note.String))
			}
			sb.WriteString("\n")
			count++
		}
		if count == 0 {
			return textResult("No annual events found.")
		}
		return textResult(sb.String())
	}
}

type AnnualEventUpdateParams struct {
	ID   int    `json:"id" jsonschema:"Annual event ID"`
	Date string `json:"date,omitempty" jsonschema:"New date"`
	Note string `json:"note,omitempty" jsonschema:"New note"`
}

func makeAnnualEventUpdate(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *AnnualEventUpdateParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *AnnualEventUpdateParams) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		var sets []string
		var args []any
		if p.Date != "" {
			if _, err := time.Parse("2006-01-02", p.Date); err != nil {
				return errResult("date must be YYYY-MM-DD")
			}
			sets = append(sets, "date = ?")
			args = append(args, p.Date)
		}
		if p.Note != "" {
			sets = append(sets, "note = ?")
			args = append(args, p.Note)
		}
		if len(sets) == 0 {
			return errResult("nothing to update")
		}
		args = append(args, p.ID)
		res, err := db.Exec("UPDATE annual_event SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
		if err != nil {
			return errResult(err.Error())
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return errResult(fmt.Sprintf("annual event with ID %d not found", p.ID))
		}
		return textResult(fmt.Sprintf("Updated annual event ID %d", p.ID))
	}
}

func makeAnnualEventDelete(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *IDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IDParam) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		res, err := db.Exec("DELETE FROM annual_event WHERE id = ?", p.ID)
		if err != nil {
			return errResult(err.Error())
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return errResult(fmt.Sprintf("annual event with ID %d not found", p.ID))
		}
		return textResult(fmt.Sprintf("Deleted annual event ID %d", p.ID))
	}
}

type AnnualEventUpcomingParams struct {
	Days int `json:"days,omitempty" jsonschema:"Look-ahead window in days (default 30)"`
}

func makeAnnualEventUpcoming(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *AnnualEventUpcomingParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *AnnualEventUpcomingParams) (*mcp.CallToolResult, any, error) {
		days := p.Days
		if days <= 0 {
			days = 30
		}

		rows, err := db.Query(`SELECT e.id, p.name, e.type, e.date, e.note
			FROM annual_event e JOIN person p ON e.person_id = p.id`)
		if err != nil {
			return errResult(err.Error())
		}
		defer rows.Close()

		todayT := time.Now().In(appLoc)
		type upcoming struct {
			id         int64
			person     string
			typ        string
			date       string
			note       sql.NullString
			next       time.Time
			yearsUntil int
		}
		var events []upcoming
		for rows.Next() {
			var u upcoming
			rows.Scan(&u.id, &u.person, &u.typ, &u.date, &u.note)
			orig, err := time.ParseInLocation("2006-01-02", u.date, appLoc)
			if err != nil {
				continue
			}
			next := time.Date(todayT.Year(), orig.Month(), orig.Day(), 0, 0, 0, 0, appLoc)
			if next.Before(todayT.Truncate(24 * time.Hour)) {
				next = next.AddDate(1, 0, 0)
			}
			u.next = next
			u.yearsUntil = next.Year() - orig.Year()
			events = append(events, u)
		}

		windowEnd := todayT.AddDate(0, 0, days)
		var filtered []upcoming
		for _, e := range events {
			if !e.next.After(windowEnd) {
				filtered = append(filtered, e)
			}
		}

		if len(filtered) == 0 {
			return textResult(fmt.Sprintf("No annual events in the next %d days.", days))
		}

		// sort by next date
		for i := 1; i < len(filtered); i++ {
			for j := i; j > 0 && filtered[j-1].next.After(filtered[j].next); j-- {
				filtered[j-1], filtered[j] = filtered[j], filtered[j-1]
			}
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Upcoming events (next %d days):\n\n", days))
		for _, e := range filtered {
			daysUntil := int(e.next.Sub(todayT.Truncate(24*time.Hour)).Hours() / 24)
			sb.WriteString(fmt.Sprintf("  [%d] %s | %s | %s (in %dd, #%d)",
				e.id, e.next.Format("2006-01-02"), e.person, e.typ, daysUntil, e.yearsUntil))
			if e.note.Valid {
				sb.WriteString(fmt.Sprintf(" | %s", e.note.String))
			}
			sb.WriteString("\n")
		}
		return textResult(sb.String())
	}
}
```

---

### Task 26: `main.go` — Tool-Registration für Person + Annual Event

**Files:**
- Modify: `cmd/personal/main.go`

- [ ] **Step 1: Vor `srv.ListenAndServe(...)` ergänzen**

```go
	// Person
	mcp.AddTool(s, &mcp.Tool{Name: "person_add", Description: "Add a new person"}, makePersonAdd(db))
	mcp.AddTool(s, &mcp.Tool{Name: "person_list", Description: "List people, optionally filtered by name substring"}, makePersonList(db))
	mcp.AddTool(s, &mcp.Tool{Name: "person_update", Description: "Update a person (name, note)"}, makePersonUpdate(db))
	mcp.AddTool(s, &mcp.Tool{Name: "person_delete", Description: "Delete a person and all their events"}, makePersonDelete(db))

	// Annual Event
	mcp.AddTool(s, &mcp.Tool{Name: "annual_event_add", Description: "Add a recurring yearly event (birthday, anniversary, name_day) for a person"}, makeAnnualEventAdd(db))
	mcp.AddTool(s, &mcp.Tool{Name: "annual_event_list", Description: "List annual events, optionally filtered by person, type, or calendar month"}, makeAnnualEventList(db))
	mcp.AddTool(s, &mcp.Tool{Name: "annual_event_update", Description: "Update an annual event"}, makeAnnualEventUpdate(db))
	mcp.AddTool(s, &mcp.Tool{Name: "annual_event_delete", Description: "Delete an annual event"}, makeAnnualEventDelete(db))
	mcp.AddTool(s, &mcp.Tool{Name: "annual_event_upcoming", Description: "List annual events in the next N days (default 30)"}, makeAnnualEventUpcoming(db))
```

---

### Task 27: API-Coverage-Doc — Person + Annual Event ergänzen

**Files:**
- Modify: `docs/api-coverage/personal.md`

- [ ] **Step 1: Neue Sektionen ans Ende vor `## Hinweise` einfügen**

```markdown
### Person

| Tool | Parameter | Status |
|------|-----------|--------|
| person_add | name, note? | implemented |
| person_list | search? | implemented |
| person_update | id, name?, note? | implemented |
| person_delete | id | implemented |

### Annual Event

| Tool | Parameter | Status |
|------|-----------|--------|
| annual_event_add | person, type, date, note? | implemented |
| annual_event_list | person?, type?, month? | implemented |
| annual_event_update | id, date?, note? | implemented |
| annual_event_delete | id | implemented |
| annual_event_upcoming | days? (default 30) | implemented |
```

---

### Task 28: Build + Commit

- [ ] **Step 1: Build**

```bash
cd ~/Projects/private/traffino/mcp-server
make personal
```
Expected: Grün.

- [ ] **Step 2: Commit + Push**

```bash
git add -A
git commit -m "$(cat <<'EOF'
feat(personal): add person and annual event domains

- person table (id, name UNIQUE, note) with CRUD tools
- annual_event table (person_id FK, type in birthday/anniversary/
  name_day, date, note, UNIQUE(person_id, type)) with CRUD plus
  annual_event_upcoming (next N days, sorted by next occurrence).
- Index on substr(date, 6) for month filtering.
- resolvePerson helper alongside resolveCompany.
- Migration v3 for the two new tables.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
git push
```

---

## Phase 4 — Project (Commit 4)

Ziel: `project` Domain mit optionaler Company-Bindung. Neue Tabelle, 5 Tools, `resolveProject` Helper.

### Task 29: Schema-Migration V4

**Files:**
- Modify: `cmd/personal/db.go`

- [ ] **Step 1: Migrations-Slice ergänzen**

```go
	migrations := []func(*sql.DB) error{
		migrateToV1,
		migrateToV2,
		migrateToV3,
		migrateToV4,
	}
```

- [ ] **Step 2: Funktion ergänzen**

```go
func migrateToV4(db *sql.DB) error {
	_, err := db.Exec(`
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
	`)
	return err
}
```

---

### Task 30: `resolveProject` in `helpers.go` ergänzen

**Files:**
- Modify: `cmd/personal/helpers.go`

- [ ] **Step 1: Funktion anfügen**

```go
// resolveProject resolves a project by name (case-insensitive) or ID.
// If companyHint is non-empty, name lookups are scoped to that company (or NULL for private).
// Returns error on ambiguity.
func resolveProject(db *sql.DB, input, companyHint string) (int64, string, error) {
	if id, err := strconv.ParseInt(input, 10, 64); err == nil {
		var name string
		err := db.QueryRow("SELECT name FROM project WHERE id = ?", id).Scan(&name)
		if err == sql.ErrNoRows {
			return 0, "", fmt.Errorf("project with ID %d not found", id)
		}
		if err != nil {
			return 0, "", err
		}
		return id, name, nil
	}

	if companyHint != "" {
		if strings.ToLower(companyHint) == "private" || companyHint == "-" {
			var id int64
			var name string
			err := db.QueryRow("SELECT id, name FROM project WHERE lower(name) = lower(?) AND company_id IS NULL", input).Scan(&id, &name)
			if err == sql.ErrNoRows {
				return 0, "", fmt.Errorf("private project %q not found", input)
			}
			if err != nil {
				return 0, "", err
			}
			return id, name, nil
		}
		companyID, _, err := resolveCompany(db, companyHint)
		if err != nil {
			return 0, "", err
		}
		var id int64
		var name string
		err = db.QueryRow("SELECT id, name FROM project WHERE lower(name) = lower(?) AND company_id = ?", input, companyID).Scan(&id, &name)
		if err == sql.ErrNoRows {
			return 0, "", fmt.Errorf("project %q not found in given company", input)
		}
		if err != nil {
			return 0, "", err
		}
		return id, name, nil
	}

	rows, err := db.Query(`SELECT p.id, p.name, COALESCE(c.name, '-') FROM project p
		LEFT JOIN company c ON p.company_id = c.id
		WHERE lower(p.name) = lower(?)`, input)
	if err != nil {
		return 0, "", err
	}
	defer rows.Close()
	type match struct {
		id      int64
		name    string
		company string
	}
	var matches []match
	for rows.Next() {
		var m match
		rows.Scan(&m.id, &m.name, &m.company)
		matches = append(matches, m)
	}
	switch len(matches) {
	case 0:
		return 0, "", fmt.Errorf("project %q not found", input)
	case 1:
		return matches[0].id, matches[0].name, nil
	default:
		var companies []string
		for _, m := range matches {
			companies = append(companies, m.company)
		}
		return 0, "", fmt.Errorf("project %q is ambiguous — exists in: %s. Use ID or pass company=<name>", input, strings.Join(companies, ", "))
	}
}
```

---

### Task 31: `project.go` — 5 Tools

**Files:**
- Create: `cmd/personal/project.go`

- [ ] **Step 1: Datei anlegen**

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ProjectAddParams struct {
	Name    string `json:"name" jsonschema:"Project name"`
	Company string `json:"company,omitempty" jsonschema:"Company name or ID; omit for private project"`
	Note    string `json:"note,omitempty" jsonschema:"Optional note"`
}

func makeProjectAdd(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *ProjectAddParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ProjectAddParams) (*mcp.CallToolResult, any, error) {
		if p.Name == "" {
			return errResult("name is required")
		}
		var companyID any = nil
		scope := "private"
		if p.Company != "" {
			id, name, err := resolveCompany(db, p.Company)
			if err != nil {
				return errResult(err.Error())
			}
			companyID = id
			scope = name
		}
		res, err := db.Exec(
			"INSERT INTO project (name, company_id, note) VALUES (?, ?, ?)",
			p.Name, companyID, nilIfEmpty(p.Note),
		)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return errResult(fmt.Sprintf("project %q already exists in %s", p.Name, scope))
			}
			return errResult(err.Error())
		}
		id, _ := res.LastInsertId()
		return textResult(fmt.Sprintf("Added project %q in %s (ID: %d)", p.Name, scope, id))
	}
}

type ProjectListParams struct {
	Company string `json:"company,omitempty" jsonschema:"Filter by company name/ID (use 'private' for private projects)"`
	Status  string `json:"status,omitempty" jsonschema:"Filter by status: active (default), archived, or all"`
}

func makeProjectList(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *ProjectListParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ProjectListParams) (*mcp.CallToolResult, any, error) {
		query := `SELECT p.id, p.name, COALESCE(c.name, '-'), p.status, p.note
			FROM project p LEFT JOIN company c ON p.company_id = c.id WHERE 1=1`
		var args []any

		if p.Company != "" {
			if strings.ToLower(p.Company) == "private" || p.Company == "-" {
				query += " AND p.company_id IS NULL"
			} else {
				companyID, _, err := resolveCompany(db, p.Company)
				if err != nil {
					return errResult(err.Error())
				}
				query += " AND p.company_id = ?"
				args = append(args, companyID)
			}
		}
		status := p.Status
		if status == "" {
			status = "active"
		}
		if status != "all" {
			if status != "active" && status != "archived" {
				return errResult("status must be one of: active, archived, all")
			}
			query += " AND p.status = ?"
			args = append(args, status)
		}
		query += " ORDER BY COALESCE(c.name, '-'), p.name"

		rows, err := db.Query(query, args...)
		if err != nil {
			return errResult(err.Error())
		}
		defer rows.Close()

		var sb strings.Builder
		sb.WriteString("Projects:\n\n")
		count := 0
		for rows.Next() {
			var id int64
			var name, company, st string
			var note sql.NullString
			rows.Scan(&id, &name, &company, &st, &note)
			sb.WriteString(fmt.Sprintf("  [%d] %s | %s | %s", id, company, name, st))
			if note.Valid {
				sb.WriteString(fmt.Sprintf(" | %s", note.String))
			}
			sb.WriteString("\n")
			count++
		}
		if count == 0 {
			return textResult("No projects found.")
		}
		return textResult(sb.String())
	}
}

type ProjectUpdateParams struct {
	ID      int    `json:"id" jsonschema:"Project ID"`
	Name    string `json:"name,omitempty" jsonschema:"New name"`
	Company string `json:"company,omitempty" jsonschema:"New company (name/ID) or 'private' to detach"`
	Status  string `json:"status,omitempty" jsonschema:"New status: active or archived"`
	Note    string `json:"note,omitempty" jsonschema:"New note"`
}

func makeProjectUpdate(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *ProjectUpdateParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ProjectUpdateParams) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		var sets []string
		var args []any
		if p.Name != "" {
			sets = append(sets, "name = ?")
			args = append(args, p.Name)
		}
		if p.Company != "" {
			if strings.ToLower(p.Company) == "private" || p.Company == "-" {
				sets = append(sets, "company_id = NULL")
			} else {
				cid, _, err := resolveCompany(db, p.Company)
				if err != nil {
					return errResult(err.Error())
				}
				sets = append(sets, "company_id = ?")
				args = append(args, cid)
			}
		}
		if p.Status != "" {
			if p.Status != "active" && p.Status != "archived" {
				return errResult("status must be active or archived")
			}
			sets = append(sets, "status = ?")
			args = append(args, p.Status)
		}
		if p.Note != "" {
			sets = append(sets, "note = ?")
			args = append(args, p.Note)
		}
		if len(sets) == 0 {
			return errResult("nothing to update")
		}
		args = append(args, p.ID)
		res, err := db.Exec("UPDATE project SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
		if err != nil {
			return errResult(err.Error())
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return errResult(fmt.Sprintf("project with ID %d not found", p.ID))
		}
		return textResult(fmt.Sprintf("Updated project ID %d", p.ID))
	}
}

func makeProjectArchive(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *IDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IDParam) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		res, err := db.Exec("UPDATE project SET status = 'archived' WHERE id = ?", p.ID)
		if err != nil {
			return errResult(err.Error())
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return errResult(fmt.Sprintf("project with ID %d not found", p.ID))
		}
		return textResult(fmt.Sprintf("Archived project ID %d", p.ID))
	}
}

func makeProjectDelete(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *IDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IDParam) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		var name string
		err := db.QueryRow("SELECT name FROM project WHERE id = ?", p.ID).Scan(&name)
		if err == sql.ErrNoRows {
			return errResult(fmt.Sprintf("project with ID %d not found", p.ID))
		}
		if err != nil {
			return errResult(err.Error())
		}
		// todo table exists from Phase 5; in Phase 4 the FK simply isn't created yet,
		// so count may be 0. We keep the count expression generic.
		var linkedTodos int
		db.QueryRow("SELECT COUNT(*) FROM todo WHERE project_id = ?", p.ID).Scan(&linkedTodos)

		if _, err := db.Exec("DELETE FROM project WHERE id = ?", p.ID); err != nil {
			return errResult(err.Error())
		}
		return textResult(fmt.Sprintf("Deleted project %q and %d linked todos (ID: %d)", name, linkedTodos, p.ID))
	}
}
```

*Hinweis zur Kompatibilität vor Phase 5: Die `SELECT COUNT(*) FROM todo ...`-Query wird fehlschlagen, solange es die `todo`-Tabelle nicht gibt. Daher defensiv: wir umklammern mit `db.QueryRow(...).Scan(&linkedTodos)`, das bei SQL-Fehler einfach `linkedTodos = 0` lässt (Scan gibt Error zurück, der ignoriert wird). Siehe Step 2.*

- [ ] **Step 2: Schutz vor fehlender todo-Tabelle**

In `makeProjectDelete`, die Zeile
```go
db.QueryRow("SELECT COUNT(*) FROM todo WHERE project_id = ?", p.ID).Scan(&linkedTodos)
```
schluckt den Fehler implizit — `Scan` gibt `no such table: todo` als Error zurück, `linkedTodos` bleibt 0. Das ist OK bis Phase 5 die Tabelle anlegt.

---

### Task 32: `main.go` — Project Tools registrieren

**Files:**
- Modify: `cmd/personal/main.go`

- [ ] **Step 1: Vor `srv.ListenAndServe(...)` ergänzen**

```go
	// Project
	mcp.AddTool(s, &mcp.Tool{Name: "project_add", Description: "Add a new project, optionally linked to a company"}, makeProjectAdd(db))
	mcp.AddTool(s, &mcp.Tool{Name: "project_list", Description: "List projects (active by default); filter by company or status"}, makeProjectList(db))
	mcp.AddTool(s, &mcp.Tool{Name: "project_update", Description: "Update a project (name, company, status, note)"}, makeProjectUpdate(db))
	mcp.AddTool(s, &mcp.Tool{Name: "project_archive", Description: "Archive a project (sets status='archived')"}, makeProjectArchive(db))
	mcp.AddTool(s, &mcp.Tool{Name: "project_delete", Description: "Delete a project and cascade-delete its todos"}, makeProjectDelete(db))
```

---

### Task 33: API-Coverage — Project-Sektion

**Files:**
- Modify: `docs/api-coverage/personal.md`

- [ ] **Step 1: Sektion vor `## Hinweise` einfügen**

```markdown
### Project

| Tool | Parameter | Status |
|------|-----------|--------|
| project_add | name, company?, note? | implemented |
| project_list | company?, status? | implemented |
| project_update | id, name?, company?, status?, note? | implemented |
| project_archive | id | implemented |
| project_delete | id | implemented |
```

---

### Task 34: Build + Commit

- [ ] **Step 1: Build**

```bash
cd ~/Projects/private/traffino/mcp-server
make personal
```
Expected: Grün.

- [ ] **Step 2: Commit + Push**

```bash
git add -A
git commit -m "$(cat <<'EOF'
feat(personal): add project domain with optional company binding

- project table (name, company_id nullable, status active/archived,
  note) with UNIQUE(company_id, name) plus partial unique index for
  company_id IS NULL to enforce unique private project names.
- 5 tools: project_add, project_list (default status=active, company
  filter supports 'private'), project_update, project_archive,
  project_delete (cascade on todos, count reported).
- resolveProject helper with ambiguity detection across companies.
- Migration v4.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
git push
```

---

## Phase 5 — Recurrence + TODO (Commit 5)

Ziel: `recurrence.go` mit TDD, `todo`/`todo_completion`-Tabellen, 9 TODO-Tools, Completion-Logic für wiederkehrende Aufgaben.

### Task 35: `recurrence_test.go` — Failing Tests

**Files:**
- Create: `cmd/personal/recurrence_test.go`

- [ ] **Step 1: Test-Datei anlegen**

```go
package main

import (
	"testing"
	"time"
)

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("bad test date %q: %v", s, err)
	}
	return d
}

func TestParsePattern(t *testing.T) {
	cases := []struct {
		in        string
		wantType  string
		wantDays  []int
		wantDay   int
		wantError bool
	}{
		{`{"type":"weekday","days":[1,3,5]}`, "weekday", []int{1, 3, 5}, 0, false},
		{`{"type":"monthday","day":15}`, "monthday", nil, 15, false},
		{`{"type":"weekday","days":[]}`, "", nil, 0, true},
		{`{"type":"weekday","days":[8]}`, "", nil, 0, true},
		{`{"type":"monthday","day":32}`, "", nil, 0, true},
		{`{"type":"monthday","day":0}`, "", nil, 0, true},
		{`{"type":"foo"}`, "", nil, 0, true},
		{`{}`, "", nil, 0, true},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			p, err := ParsePattern(c.in)
			if c.wantError {
				if err == nil {
					t.Fatalf("expected error for %q, got %+v", c.in, p)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.Type != c.wantType {
				t.Errorf("Type = %q, want %q", p.Type, c.wantType)
			}
			if c.wantType == "weekday" {
				if len(p.Days) != len(c.wantDays) {
					t.Fatalf("Days length %d, want %d", len(p.Days), len(c.wantDays))
				}
				for i, d := range p.Days {
					if d != c.wantDays[i] {
						t.Errorf("Days[%d] = %d, want %d", i, d, c.wantDays[i])
					}
				}
			}
			if c.wantType == "monthday" && p.Day != c.wantDay {
				t.Errorf("Day = %d, want %d", p.Day, c.wantDay)
			}
		})
	}
}

func TestNextOccurrenceWeekday(t *testing.T) {
	// Mon=1..Sun=7
	p := Pattern{Type: "weekday", Days: []int{1}} // Monday only

	cases := []struct {
		from, want string
	}{
		{"2026-04-13", "2026-04-20"}, // Mon -> next Mon
		{"2026-04-15", "2026-04-20"}, // Wed -> next Mon
		{"2026-04-19", "2026-04-20"}, // Sun -> next Mon
		{"2026-04-20", "2026-04-27"}, // Mon -> strictly after -> next Mon
	}
	for _, c := range cases {
		t.Run(c.from, func(t *testing.T) {
			got, err := p.NextOccurrence(mustDate(t, c.from))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			gotS := got.Format("2006-01-02")
			if gotS != c.want {
				t.Errorf("NextOccurrence(%s) = %s, want %s", c.from, gotS, c.want)
			}
		})
	}
}

func TestNextOccurrenceWeekdayMultiple(t *testing.T) {
	p := Pattern{Type: "weekday", Days: []int{1, 3, 5}} // Mon, Wed, Fri
	got, _ := p.NextOccurrence(mustDate(t, "2026-04-13")) // Mon
	if got.Format("2006-01-02") != "2026-04-15" {         // next = Wed
		t.Errorf("got %s, want 2026-04-15", got.Format("2006-01-02"))
	}
	got, _ = p.NextOccurrence(mustDate(t, "2026-04-18")) // Sat
	if got.Format("2006-01-02") != "2026-04-20" {        // next Mon
		t.Errorf("got %s, want 2026-04-20", got.Format("2006-01-02"))
	}
}

func TestNextOccurrenceMonthday(t *testing.T) {
	p := Pattern{Type: "monthday", Day: 15}
	cases := []struct{ from, want string }{
		{"2026-04-10", "2026-04-15"},
		{"2026-04-15", "2026-05-15"}, // strict "after"
		{"2026-04-16", "2026-05-15"},
	}
	for _, c := range cases {
		got, _ := p.NextOccurrence(mustDate(t, c.from))
		if got.Format("2006-01-02") != c.want {
			t.Errorf("from %s: got %s, want %s", c.from, got.Format("2006-01-02"), c.want)
		}
	}
}

func TestNextOccurrenceMonthdayOverflow(t *testing.T) {
	p := Pattern{Type: "monthday", Day: 31}
	// After Jan 31: next is Feb last day (28 in 2026)
	got, _ := p.NextOccurrence(mustDate(t, "2026-01-31"))
	if got.Format("2006-01-02") != "2026-02-28" {
		t.Errorf("got %s, want 2026-02-28", got.Format("2006-01-02"))
	}
	// After Feb 28: next is Mar 31
	got, _ = p.NextOccurrence(mustDate(t, "2026-02-28"))
	if got.Format("2006-01-02") != "2026-03-31" {
		t.Errorf("got %s, want 2026-03-31", got.Format("2006-01-02"))
	}
}

func TestPatternSerialize(t *testing.T) {
	p := Pattern{Type: "weekday", Days: []int{1, 3, 5}}
	s, err := p.Serialize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	round, _ := ParsePattern(s)
	if round.Type != "weekday" || len(round.Days) != 3 {
		t.Errorf("roundtrip failed: %+v", round)
	}
}
```

- [ ] **Step 2: Tests laufen lassen — rot**

```bash
cd ~/Projects/private/traffino/mcp-server
go test ./cmd/personal/ -run 'TestParsePattern|TestNextOccurrence|TestPatternSerialize' -v
```
Expected: FAIL mit `undefined: ParsePattern`, `undefined: Pattern`.

---

### Task 36: `recurrence.go` implementieren

**Files:**
- Create: `cmd/personal/recurrence.go`

- [ ] **Step 1: Datei anlegen**

```go
package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

type Pattern struct {
	Type string `json:"type"`
	Days []int  `json:"days,omitempty"` // weekday: 1=Mon..7=Sun
	Day  int    `json:"day,omitempty"`  // monthday: 1..31
}

func ParsePattern(s string) (Pattern, error) {
	var p Pattern
	if err := json.Unmarshal([]byte(s), &p); err != nil {
		return Pattern{}, fmt.Errorf("invalid recurrence JSON: %w", err)
	}
	switch p.Type {
	case "weekday":
		if len(p.Days) == 0 {
			return Pattern{}, fmt.Errorf("weekday pattern requires non-empty days (1..7, Mon=1)")
		}
		for _, d := range p.Days {
			if d < 1 || d > 7 {
				return Pattern{}, fmt.Errorf("weekday days must be 1..7, got %d", d)
			}
		}
		sort.Ints(p.Days)
	case "monthday":
		if p.Day < 1 || p.Day > 31 {
			return Pattern{}, fmt.Errorf("monthday day must be 1..31, got %d", p.Day)
		}
	default:
		return Pattern{}, fmt.Errorf("pattern type must be 'weekday' or 'monthday', got %q", p.Type)
	}
	return p, nil
}

func (p Pattern) Serialize() (string, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// NextOccurrence returns the next date strictly after `from` matching the pattern.
// `from` should be a date at midnight (time portion ignored).
func (p Pattern) NextOccurrence(from time.Time) (time.Time, error) {
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	switch p.Type {
	case "weekday":
		return nextWeekday(p.Days, from), nil
	case "monthday":
		return nextMonthday(p.Day, from), nil
	}
	return time.Time{}, fmt.Errorf("unknown pattern type %q", p.Type)
}

func isoWeekday(t time.Time) int {
	// time.Weekday: Sunday=0..Saturday=6. Convert to ISO: Mon=1..Sun=7.
	w := int(t.Weekday())
	if w == 0 {
		return 7
	}
	return w
}

func nextWeekday(days []int, from time.Time) time.Time {
	for offset := 1; offset <= 7; offset++ {
		candidate := from.AddDate(0, 0, offset)
		cw := isoWeekday(candidate)
		for _, d := range days {
			if d == cw {
				return candidate
			}
		}
	}
	return from.AddDate(0, 0, 7) // unreachable given validation
}

func nextMonthday(day int, from time.Time) time.Time {
	year, month := from.Year(), from.Month()
	// Candidate this month (if not already past):
	if from.Day() < day {
		target := clampMonthday(year, month, day, from.Location())
		if target.After(from) {
			return target
		}
	}
	// Advance one month
	month++
	if month > 12 {
		month = 1
		year++
	}
	return clampMonthday(year, month, day, from.Location())
}

func clampMonthday(year int, month time.Month, day int, loc *time.Location) time.Time {
	// Last day of month = day 0 of next month
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}
```

- [ ] **Step 2: Tests laufen lassen — grün**

```bash
go test ./cmd/personal/ -run 'TestParsePattern|TestNextOccurrence|TestPatternSerialize' -v
```
Expected: Alle PASS.

---

### Task 37: Schema-Migration V5 (todo + todo_completion)

**Files:**
- Modify: `cmd/personal/db.go`

- [ ] **Step 1: Migrations-Slice ergänzen**

```go
	migrations := []func(*sql.DB) error{
		migrateToV1,
		migrateToV2,
		migrateToV3,
		migrateToV4,
		migrateToV5,
	}
```

- [ ] **Step 2: Funktion ergänzen**

```go
func migrateToV5(db *sql.DB) error {
	_, err := db.Exec(`
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
```

---

### Task 38: `todo.go` — Tools für Add/List/Complete (erste Batch)

**Files:**
- Create: `cmd/personal/todo.go`

- [ ] **Step 1: Datei anlegen mit Add-Tools + helpers**

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// patternFromParams builds a Pattern from structured params.
// Returns (pattern, isSet, error). If recurrence_type is empty, isSet=false.
func patternFromParams(recType string, recDays []int, recDay int) (Pattern, bool, error) {
	if recType == "" {
		if len(recDays) > 0 || recDay > 0 {
			return Pattern{}, false, fmt.Errorf("recurrence_type is required when recurrence_days or recurrence_day is set")
		}
		return Pattern{}, false, nil
	}
	p := Pattern{Type: recType, Days: recDays, Day: recDay}
	serialized, err := p.Serialize()
	if err != nil {
		return Pattern{}, false, err
	}
	// Re-parse to validate
	if _, err := ParsePattern(serialized); err != nil {
		return Pattern{}, false, err
	}
	return p, true, nil
}

type TodoAddParams struct {
	Title       string `json:"title" jsonschema:"TODO title"`
	Project     string `json:"project,omitempty" jsonschema:"Project name or ID"`
	Company     string `json:"company,omitempty" jsonschema:"Company name or ID (if no project)"`
	DueDate     string `json:"due_date,omitempty" jsonschema:"Due date YYYY-MM-DD"`
	Note        string `json:"note,omitempty" jsonschema:"Optional note"`
	RecurrenceType string `json:"recurrence_type,omitempty" jsonschema:"Recurrence type: weekday or monthday"`
	RecurrenceDays []int  `json:"recurrence_days,omitempty" jsonschema:"Weekday ISO numbers 1..7 for recurrence_type=weekday"`
	RecurrenceDay  int    `json:"recurrence_day,omitempty" jsonschema:"Day of month 1..31 for recurrence_type=monthday"`
}

func makeTodoAdd(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *TodoAddParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *TodoAddParams) (*mcp.CallToolResult, any, error) {
		if p.Title == "" {
			return errResult("title is required")
		}
		var projectID, companyID any = nil, nil
		if p.Project != "" {
			id, _, err := resolveProject(db, p.Project, p.Company)
			if err != nil {
				return errResult(err.Error())
			}
			projectID = id
		} else if p.Company != "" {
			id, _, err := resolveCompany(db, p.Company)
			if err != nil {
				return errResult(err.Error())
			}
			companyID = id
		}
		pattern, hasPattern, err := patternFromParams(p.RecurrenceType, p.RecurrenceDays, p.RecurrenceDay)
		if err != nil {
			return errResult(err.Error())
		}
		var patternJSON any = nil
		if hasPattern {
			s, _ := pattern.Serialize()
			patternJSON = s
		}
		res, err := db.Exec(
			`INSERT INTO todo (title, project_id, company_id, due_date, note, recurrence_pattern)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			p.Title, projectID, companyID, nilIfEmpty(p.DueDate), nilIfEmpty(p.Note), patternJSON,
		)
		if err != nil {
			return errResult(err.Error())
		}
		id, _ := res.LastInsertId()
		return textResult(fmt.Sprintf("Added todo %q (ID: %d)", p.Title, id))
	}
}

type SubtaskAddParams struct {
	ParentID int    `json:"parent_id" jsonschema:"Top-level TODO ID to attach the subtask to"`
	Title    string `json:"title" jsonschema:"Subtask title"`
	DueDate  string `json:"due_date,omitempty" jsonschema:"Due date YYYY-MM-DD"`
	Note     string `json:"note,omitempty" jsonschema:"Optional note"`
}

func makeSubtaskAdd(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *SubtaskAddParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *SubtaskAddParams) (*mcp.CallToolResult, any, error) {
		if p.ParentID == 0 || p.Title == "" {
			return errResult("parent_id and title are required")
		}
		var parentParent sql.NullInt64
		err := db.QueryRow("SELECT parent_id FROM todo WHERE id = ?", p.ParentID).Scan(&parentParent)
		if err == sql.ErrNoRows {
			return errResult(fmt.Sprintf("parent TODO with ID %d not found", p.ParentID))
		}
		if err != nil {
			return errResult(err.Error())
		}
		if parentParent.Valid {
			return errResult(fmt.Sprintf("parent ID %d is already a subtask; subtasks cannot have children (max 3 levels)", p.ParentID))
		}
		res, err := db.Exec(
			`INSERT INTO todo (title, parent_id, due_date, note) VALUES (?, ?, ?, ?)`,
			p.Title, p.ParentID, nilIfEmpty(p.DueDate), nilIfEmpty(p.Note),
		)
		if err != nil {
			return errResult(err.Error())
		}
		id, _ := res.LastInsertId()
		return textResult(fmt.Sprintf("Added subtask %q under TODO %d (ID: %d)", p.Title, p.ParentID, id))
	}
}

// completeTodo writes the completion audit and, if recurring, advances due_date.
// Called by makeTodoComplete and from makeTodoUpdate when status transitions to 'done'.
func completeTodo(db *sql.DB, id int) error {
	var dueDate, patternJSON sql.NullString
	var parentID sql.NullInt64
	err := db.QueryRow(
		"SELECT due_date, recurrence_pattern, parent_id FROM todo WHERE id = ?", id,
	).Scan(&dueDate, &patternJSON, &parentID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("todo with ID %d not found", id)
	}
	if err != nil {
		return err
	}

	now := nowString()
	var dueAtCompletion any = nil
	if dueDate.Valid {
		dueAtCompletion = dueDate.String
	}

	if _, err := db.Exec(
		"INSERT INTO todo_completion (todo_id, completed_at, due_date_at_completion) VALUES (?, ?, ?)",
		id, now, dueAtCompletion,
	); err != nil {
		return err
	}

	if patternJSON.Valid && !parentID.Valid {
		p, err := ParsePattern(patternJSON.String)
		if err != nil {
			return fmt.Errorf("invalid stored recurrence pattern: %w", err)
		}
		cursor := today()
		if dueDate.Valid && dueDate.String > cursor {
			cursor = dueDate.String
		}
		cursorT, err := parseDateLocal(cursor)
		if err != nil {
			return err
		}
		next, err := p.NextOccurrence(cursorT)
		if err != nil {
			return err
		}
		_, err = db.Exec(
			`UPDATE todo SET due_date = ?, status = 'open', completed_at = NULL WHERE id = ?`,
			next.Format("2006-01-02"), id,
		)
		return err
	}

	_, err = db.Exec(
		`UPDATE todo SET status = 'done', completed_at = ? WHERE id = ?`,
		now, id,
	)
	return err
}

func makeTodoComplete(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *IDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IDParam) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		if err := completeTodo(db, p.ID); err != nil {
			return errResult(err.Error())
		}
		return textResult(fmt.Sprintf("Completed todo ID %d", p.ID))
	}
}
```

- [ ] **Step 2: `time.go` um `parseDateLocal` ergänzen**

Am Dateiende:

```go
func parseDateLocal(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, appLoc)
}
```

---

### Task 39: `todo.go` — Rest der Tools (List, Update, Delete, Move, Upcoming, Overdue)

**Files:**
- Modify: `cmd/personal/todo.go`

- [ ] **Step 1: An das Ende von `todo.go` anhängen**

```go
type TodoListParams struct {
	Project         string `json:"project,omitempty" jsonschema:"Filter by project name/ID"`
	Company         string `json:"company,omitempty" jsonschema:"Filter by company name/ID"`
	Status          string `json:"status,omitempty" jsonschema:"Filter by status; 'all' for every status; default: open+in_progress"`
	DueBefore       string `json:"due_before,omitempty" jsonschema:"Only todos with due_date < this date"`
	DueAfter        string `json:"due_after,omitempty" jsonschema:"Only todos with due_date > this date"`
	IncludeSubtasks bool   `json:"include_subtasks,omitempty" jsonschema:"Render subtasks indented beneath parents"`
}

func makeTodoList(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *TodoListParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *TodoListParams) (*mcp.CallToolResult, any, error) {
		query := `SELECT t.id, t.title, COALESCE(pr.name,'-'), COALESCE(c.name,'-'), t.status, COALESCE(t.due_date,'-'), t.recurrence_pattern, t.parent_id
			FROM todo t
			LEFT JOIN project pr ON t.project_id = pr.id
			LEFT JOIN company c  ON t.company_id = c.id
			WHERE t.parent_id IS NULL`
		var args []any

		if p.Project != "" {
			pid, _, err := resolveProject(db, p.Project, p.Company)
			if err != nil {
				return errResult(err.Error())
			}
			query += " AND t.project_id = ?"
			args = append(args, pid)
		} else if p.Company != "" {
			cid, _, err := resolveCompany(db, p.Company)
			if err != nil {
				return errResult(err.Error())
			}
			query += " AND t.company_id = ?"
			args = append(args, cid)
		}

		switch p.Status {
		case "":
			query += " AND t.status IN ('open','in_progress')"
		case "all":
			// no status filter
		case "open", "in_progress", "waiting", "done", "cancelled":
			query += " AND t.status = ?"
			args = append(args, p.Status)
		default:
			return errResult("status must be one of: open, in_progress, waiting, done, cancelled, all")
		}

		if p.DueBefore != "" {
			query += " AND t.due_date < ?"
			args = append(args, p.DueBefore)
		}
		if p.DueAfter != "" {
			query += " AND t.due_date > ?"
			args = append(args, p.DueAfter)
		}
		query += " ORDER BY COALESCE(t.due_date, '9999-12-31'), t.id"

		rows, err := db.Query(query, args...)
		if err != nil {
			return errResult(err.Error())
		}
		defer rows.Close()

		type row struct {
			id       int64
			title    string
			project  string
			company  string
			status   string
			due      string
			pattern  sql.NullString
		}
		var items []row
		for rows.Next() {
			var r row
			var parentID sql.NullInt64
			rows.Scan(&r.id, &r.title, &r.project, &r.company, &r.status, &r.due, &r.pattern, &parentID)
			items = append(items, r)
		}

		if len(items) == 0 {
			return textResult("No todos found.")
		}

		var sb strings.Builder
		sb.WriteString("Todos:\n\n")
		for _, r := range items {
			sb.WriteString(fmt.Sprintf("  [%d] %s | %s / %s | %s", r.id, r.title, r.company, r.project, r.status))
			if r.due != "-" {
				sb.WriteString(fmt.Sprintf(" | due %s", r.due))
			}
			if r.pattern.Valid {
				sb.WriteString(" | recurring")
			}
			sb.WriteString("\n")
			if p.IncludeSubtasks {
				subRows, err := db.Query("SELECT id, title, status, COALESCE(due_date,'-') FROM todo WHERE parent_id = ? ORDER BY id", r.id)
				if err == nil {
					for subRows.Next() {
						var sid int64
						var stitle, sstatus, sdue string
						subRows.Scan(&sid, &stitle, &sstatus, &sdue)
						sb.WriteString(fmt.Sprintf("    [%d] %s | %s", sid, stitle, sstatus))
						if sdue != "-" {
							sb.WriteString(fmt.Sprintf(" | due %s", sdue))
						}
						sb.WriteString("\n")
					}
					subRows.Close()
				}
			}
		}
		return textResult(sb.String())
	}
}

type TodoUpdateParams struct {
	ID              int    `json:"id" jsonschema:"TODO ID"`
	Title           string `json:"title,omitempty" jsonschema:"New title"`
	DueDate         string `json:"due_date,omitempty" jsonschema:"New due date YYYY-MM-DD"`
	Note            string `json:"note,omitempty" jsonschema:"New note"`
	Status          string `json:"status,omitempty" jsonschema:"New status"`
	Project         string `json:"project,omitempty" jsonschema:"New project (name/ID) — top-level only"`
	Company         string `json:"company,omitempty" jsonschema:"New company (name/ID) — top-level only"`
	RecurrenceType  string `json:"recurrence_type,omitempty" jsonschema:"weekday or monthday"`
	RecurrenceDays  []int  `json:"recurrence_days,omitempty" jsonschema:"Weekday ISO numbers"`
	RecurrenceDay   int    `json:"recurrence_day,omitempty" jsonschema:"Day of month"`
	ClearRecurrence bool   `json:"clear_recurrence,omitempty" jsonschema:"Set true to remove recurrence"`
}

func makeTodoUpdate(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *TodoUpdateParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *TodoUpdateParams) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		var parentID sql.NullInt64
		err := db.QueryRow("SELECT parent_id FROM todo WHERE id = ?", p.ID).Scan(&parentID)
		if err == sql.ErrNoRows {
			return errResult(fmt.Sprintf("todo with ID %d not found", p.ID))
		}
		if err != nil {
			return errResult(err.Error())
		}
		isSubtask := parentID.Valid

		if isSubtask && (p.Project != "" || p.Company != "" ||
			p.RecurrenceType != "" || len(p.RecurrenceDays) > 0 || p.RecurrenceDay > 0 || p.ClearRecurrence) {
			return errResult("project, company, and recurrence can only be set on top-level todos")
		}

		var sets []string
		var args []any
		if p.Title != "" {
			sets = append(sets, "title = ?")
			args = append(args, p.Title)
		}
		if p.DueDate != "" {
			sets = append(sets, "due_date = ?")
			args = append(args, p.DueDate)
		}
		if p.Note != "" {
			sets = append(sets, "note = ?")
			args = append(args, p.Note)
		}
		if p.Project != "" {
			pid, _, err := resolveProject(db, p.Project, p.Company)
			if err != nil {
				return errResult(err.Error())
			}
			sets = append(sets, "project_id = ?", "company_id = NULL")
			args = append(args, pid)
		} else if p.Company != "" {
			cid, _, err := resolveCompany(db, p.Company)
			if err != nil {
				return errResult(err.Error())
			}
			sets = append(sets, "company_id = ?", "project_id = NULL")
			args = append(args, cid)
		}
		if p.ClearRecurrence {
			sets = append(sets, "recurrence_pattern = NULL")
		} else if p.RecurrenceType != "" {
			pat, hasPat, err := patternFromParams(p.RecurrenceType, p.RecurrenceDays, p.RecurrenceDay)
			if err != nil {
				return errResult(err.Error())
			}
			if hasPat {
				s, _ := pat.Serialize()
				sets = append(sets, "recurrence_pattern = ?")
				args = append(args, s)
			}
		}

		if p.Status != "" {
			// Completion via status transition: handle via completeTodo
			if p.Status == "done" {
				// apply any pending sets first (title/due/note)
				if len(sets) > 0 {
					args = append(args, p.ID)
					if _, err := db.Exec("UPDATE todo SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...); err != nil {
						return errResult(err.Error())
					}
				}
				if err := completeTodo(db, p.ID); err != nil {
					return errResult(err.Error())
				}
				return textResult(fmt.Sprintf("Completed todo ID %d", p.ID))
			}
			valid := map[string]bool{"open": true, "in_progress": true, "waiting": true, "cancelled": true}
			if !valid[p.Status] {
				return errResult("status must be one of: open, in_progress, waiting, done, cancelled")
			}
			sets = append(sets, "status = ?", "completed_at = NULL")
			args = append(args, p.Status)
		}

		if len(sets) == 0 {
			return errResult("nothing to update")
		}
		args = append(args, p.ID)
		res, err := db.Exec("UPDATE todo SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...)
		if err != nil {
			return errResult(err.Error())
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return errResult(fmt.Sprintf("todo with ID %d not found", p.ID))
		}
		return textResult(fmt.Sprintf("Updated todo ID %d", p.ID))
	}
}

func makeTodoDelete(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *IDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IDParam) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		res, err := db.Exec("DELETE FROM todo WHERE id = ?", p.ID)
		if err != nil {
			return errResult(err.Error())
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return errResult(fmt.Sprintf("todo with ID %d not found", p.ID))
		}
		return textResult(fmt.Sprintf("Deleted todo ID %d (including subtasks and completion history)", p.ID))
	}
}

type TodoMoveParams struct {
	ID      int    `json:"id" jsonschema:"TODO ID (top-level only)"`
	Project string `json:"project,omitempty" jsonschema:"New project (name/ID); omit to clear"`
	Company string `json:"company,omitempty" jsonschema:"New company (name/ID); omit to clear"`
}

func makeTodoMove(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *TodoMoveParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *TodoMoveParams) (*mcp.CallToolResult, any, error) {
		if p.ID == 0 {
			return errResult("id is required")
		}
		var parentID sql.NullInt64
		if err := db.QueryRow("SELECT parent_id FROM todo WHERE id = ?", p.ID).Scan(&parentID); err != nil {
			if err == sql.ErrNoRows {
				return errResult(fmt.Sprintf("todo with ID %d not found", p.ID))
			}
			return errResult(err.Error())
		}
		if parentID.Valid {
			return errResult("subtasks cannot be moved; they inherit project/company from parent")
		}
		var projectID, companyID any = nil, nil
		if p.Project != "" {
			pid, _, err := resolveProject(db, p.Project, p.Company)
			if err != nil {
				return errResult(err.Error())
			}
			projectID = pid
		} else if p.Company != "" {
			cid, _, err := resolveCompany(db, p.Company)
			if err != nil {
				return errResult(err.Error())
			}
			companyID = cid
		}
		res, err := db.Exec("UPDATE todo SET project_id = ?, company_id = ? WHERE id = ?", projectID, companyID, p.ID)
		if err != nil {
			return errResult(err.Error())
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return errResult(fmt.Sprintf("todo with ID %d not found", p.ID))
		}
		return textResult(fmt.Sprintf("Moved todo ID %d", p.ID))
	}
}

type TodoUpcomingParams struct {
	Days    int    `json:"days,omitempty" jsonschema:"Look-ahead window in days (default 7)"`
	Company string `json:"company,omitempty" jsonschema:"Filter by company"`
}

func makeTodoUpcoming(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *TodoUpcomingParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *TodoUpcomingParams) (*mcp.CallToolResult, any, error) {
		days := p.Days
		if days <= 0 {
			days = 7
		}
		todayStr := today()
		cutoffT, _ := parseDateLocal(todayStr)
		cutoff := cutoffT.AddDate(0, 0, days).Format("2006-01-02")

		query := `SELECT t.id, t.title, COALESCE(pr.name,'-'), COALESCE(c.name,'-'), t.status, t.due_date
			FROM todo t
			LEFT JOIN project pr ON t.project_id = pr.id
			LEFT JOIN company c  ON t.company_id = c.id
			WHERE t.parent_id IS NULL
			  AND t.status IN ('open','in_progress')
			  AND t.due_date IS NOT NULL
			  AND t.due_date >= ?
			  AND t.due_date <= ?`
		args := []any{todayStr, cutoff}
		if p.Company != "" {
			cid, _, err := resolveCompany(db, p.Company)
			if err != nil {
				return errResult(err.Error())
			}
			query += " AND t.company_id = ?"
			args = append(args, cid)
		}
		query += " ORDER BY t.due_date, t.id"

		rows, err := db.Query(query, args...)
		if err != nil {
			return errResult(err.Error())
		}
		defer rows.Close()

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Upcoming todos (next %d days):\n\n", days))
		count := 0
		for rows.Next() {
			var id int64
			var title, project, company, status, due string
			rows.Scan(&id, &title, &project, &company, &status, &due)
			sb.WriteString(fmt.Sprintf("  [%d] %s | %s / %s | %s | due %s\n", id, title, company, project, status, due))
			count++
		}
		if count == 0 {
			return textResult(fmt.Sprintf("No upcoming todos in the next %d days.", days))
		}
		return textResult(sb.String())
	}
}

type TodoOverdueParams struct {
	Company string `json:"company,omitempty" jsonschema:"Filter by company"`
}

func makeTodoOverdue(db *sql.DB) func(context.Context, *mcp.CallToolRequest, *TodoOverdueParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *TodoOverdueParams) (*mcp.CallToolResult, any, error) {
		todayStr := today()
		query := `SELECT t.id, t.title, COALESCE(pr.name,'-'), COALESCE(c.name,'-'), t.status, t.due_date
			FROM todo t
			LEFT JOIN project pr ON t.project_id = pr.id
			LEFT JOIN company c  ON t.company_id = c.id
			WHERE t.parent_id IS NULL
			  AND t.status IN ('open','in_progress')
			  AND t.due_date IS NOT NULL
			  AND t.due_date < ?`
		args := []any{todayStr}
		if p.Company != "" {
			cid, _, err := resolveCompany(db, p.Company)
			if err != nil {
				return errResult(err.Error())
			}
			query += " AND t.company_id = ?"
			args = append(args, cid)
		}
		query += " ORDER BY t.due_date, t.id"

		rows, err := db.Query(query, args...)
		if err != nil {
			return errResult(err.Error())
		}
		defer rows.Close()

		var sb strings.Builder
		sb.WriteString("Overdue todos:\n\n")
		count := 0
		for rows.Next() {
			var id int64
			var title, project, company, status, due string
			rows.Scan(&id, &title, &project, &company, &status, &due)
			sb.WriteString(fmt.Sprintf("  [%d] %s | %s / %s | %s | due %s (overdue)\n", id, title, company, project, status, due))
			count++
		}
		if count == 0 {
			return textResult("No overdue todos.")
		}
		return textResult(sb.String())
	}
}
```

---

### Task 40: `main.go` — TODO-Tools registrieren

**Files:**
- Modify: `cmd/personal/main.go`

- [ ] **Step 1: Vor `srv.ListenAndServe(...)` ergänzen**

```go
	// TODO
	mcp.AddTool(s, &mcp.Tool{Name: "todo_add", Description: "Add a top-level TODO. Can attach to project or company, set due date, note, and recurrence (weekday or monthday)."}, makeTodoAdd(db))
	mcp.AddTool(s, &mcp.Tool{Name: "subtask_add", Description: "Add a subtask under a top-level TODO (max 3 levels)"}, makeSubtaskAdd(db))
	mcp.AddTool(s, &mcp.Tool{Name: "todo_list", Description: "List todos with filters. Default: only open/in_progress top-level todos."}, makeTodoList(db))
	mcp.AddTool(s, &mcp.Tool{Name: "todo_update", Description: "Update a todo. Setting status='done' triggers completion (audit log + recurrence advance)."}, makeTodoUpdate(db))
	mcp.AddTool(s, &mcp.Tool{Name: "todo_complete", Description: "Mark a todo as done. Convenience alias for todo_update with status='done'."}, makeTodoComplete(db))
	mcp.AddTool(s, &mcp.Tool{Name: "todo_delete", Description: "Delete a todo (cascades to subtasks and completion history)"}, makeTodoDelete(db))
	mcp.AddTool(s, &mcp.Tool{Name: "todo_move", Description: "Move a top-level todo to a different project or company"}, makeTodoMove(db))
	mcp.AddTool(s, &mcp.Tool{Name: "todo_upcoming", Description: "List open/in_progress todos due in the next N days (default 7)"}, makeTodoUpcoming(db))
	mcp.AddTool(s, &mcp.Tool{Name: "todo_overdue", Description: "List open/in_progress todos with due_date in the past"}, makeTodoOverdue(db))
```

---

### Task 41: API-Coverage — TODO-Sektion

**Files:**
- Modify: `docs/api-coverage/personal.md`

- [ ] **Step 1: Sektion ergänzen**

```markdown
### TODO

| Tool | Parameter | Status |
|------|-----------|--------|
| todo_add | title, project?, company?, due_date?, note?, recurrence_type?, recurrence_days?, recurrence_day? | implemented |
| subtask_add | parent_id, title, due_date?, note? | implemented |
| todo_list | project?, company?, status?, due_before?, due_after?, include_subtasks? | implemented |
| todo_update | id, title?, due_date?, note?, status?, project?, company?, recurrence_*?, clear_recurrence? | implemented |
| todo_complete | id | implemented |
| todo_delete | id | implemented |
| todo_move | id, project?, company? | implemented |
| todo_upcoming | days?, company? | implemented |
| todo_overdue | company? | implemented |
```

Und im Block `## Hinweise` nachschieben:

```markdown
- TODO-Hierarchie hat strikt drei Ebenen: Projekt → Todo → Subtask
- Subtasks erben project_id/company_id implizit vom Parent
- Rekurrenz nur auf Top-Level-TODOs; Pattern fest terminiert (nächste Instanz = Pattern-basiert, unabhängig von Completion-Zeitpunkt)
- `todo_list` Default-Filter: `status IN ('open','in_progress')`; `status='all'` zeigt alles
```

---

### Task 42: Build + Tests + Commit

- [ ] **Step 1: Build**

```bash
cd ~/Projects/private/traffino/mcp-server
make personal
```
Expected: Grün.

- [ ] **Step 2: Alle Tests**

```bash
make test
```
Expected: `TestComputeHours`, `TestParseTimeOfDay`, `TestParsePattern`, `TestNextOccurrence*`, `TestPatternSerialize` alle PASS.

- [ ] **Step 3: Commit + Push**

```bash
git add -A
git commit -m "$(cat <<'EOF'
feat(personal): add recurrence and three-level TODO hierarchy

- recurrence.go: Pattern (weekday/monthday), ParsePattern, Serialize,
  NextOccurrence with month-overflow clamping. Fully tested.
- todo table with project/company/parent_id references, recurrence_pattern,
  completed_at, strict 3-level hierarchy (parent of a subtask must be
  top-level) enforced in application code.
- todo_completion audit table, written on every status transition to
  'done' (including recurring todos before due_date is advanced).
- 9 tools: todo_add, subtask_add, todo_list (default open/in_progress,
  subtask indentation optional), todo_update (done-transition triggers
  completion logic), todo_complete, todo_delete, todo_move, todo_upcoming,
  todo_overdue.
- Migration v5.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
git push
```

---

## Phase 6 — Infrastructure-home Deployment

Ziel: Docker-Compose-Service `tracker` → `personal` umbenannt, Image-Tag angepasst, `PERSONAL_TZ=Europe/Berlin` + `PERSONAL_DB_PATH` im Compose-File.

### Task 43: Infrastructure-home Repo lokalisieren und Stand prüfen

**Files:**
- Reference: `~/Projects/private/traffino/infrastructure-home`

- [ ] **Step 1: Repo-Stand prüfen**

```bash
cd ~/Projects/private/traffino/infrastructure-home
git status
git pull --ff-only
```
Expected: Clean, up-to-date.

- [ ] **Step 2: tracker-Referenzen finden**

```bash
grep -rn "tracker" --include="*.yml" --include="*.yaml" --include="*.env*" --include="Makefile" .
```

Note: Konkrete Dateinamen und Zeilennummern variieren. Merke dir die Fundstellen; die nächsten Steps passen jede davon an.

---

### Task 44: Docker-Compose-Service anpassen

**Files:**
- Modify: `docker-compose.yml` (oder äquivalente Compose-Datei) in `infrastructure-home`

- [ ] **Step 1: Service-Block `tracker:` in `personal:` umbenennen**

Typischer alter Block:
```yaml
  tracker:
    image: traffino/mcp-tracker:latest
    volumes:
      - ./data/tracker:/data
    environment:
      TRACKER_DB_PATH: /data/tracker.db
    restart: unless-stopped
```

Ersatz:
```yaml
  personal:
    image: traffino/mcp-personal:latest
    volumes:
      - ./data/personal:/data
    environment:
      PERSONAL_DB_PATH: /data/personal.db
      PERSONAL_TZ: Europe/Berlin
    restart: unless-stopped
```

- [ ] **Step 2: Weitere `tracker`-Referenzen** (Reverse-Proxy-Route, `.env`-Templates, Watchtower-Labels) entsprechend anpassen — jeweils `tracker` → `personal`, `mcp-tracker` → `mcp-personal`.

- [ ] **Step 3: Datenverzeichnis** — das alte `data/tracker/` existiert mit tracker.db. Da keine Migration nötig ist, anlegen von `data/personal/` als leerem Ordner. Der alte Ordner bleibt unangetastet liegen (User kann später entfernen).

```bash
mkdir -p data/personal
```

---

### Task 45: Commit + Push (infrastructure-home)

- [ ] **Step 1: Commit**

```bash
cd ~/Projects/private/traffino/infrastructure-home
git add -A
git commit -m "$(cat <<'EOF'
refactor(mcp): rename tracker service to personal

Match the mcp-server monorepo rename:
- service name tracker -> personal
- image traffino/mcp-tracker -> traffino/mcp-personal
- env TRACKER_DB_PATH -> PERSONAL_DB_PATH (/data/personal.db)
- add PERSONAL_TZ=Europe/Berlin
- separate data volume at data/personal/

Old data/tracker/ folder is left in place; remove manually once the
service is verified.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
git push
```

---

## Self-Review (ausgeführt vor Handoff)

**1. Spec coverage:**

| Spec-Anforderung              | Task(s)          |
|-------------------------------|------------------|
| Rename + Split Datei-Layout   | 1–13             |
| PERSONAL_TZ / PERSONAL_DB_PATH| 3, 9             |
| Schema V1 (bestehend)         | 17 (migrateToV1) |
| Overtime V2 mit type+times    | 17 (migrateToV2), 18, 19 |
| ComputeHours mit Tests        | 15, 16           |
| Person + Annual Event         | 22–26            |
| Project + resolveProject      | 29–32            |
| Recurrence TDD                | 35, 36           |
| todo + todo_completion        | 37               |
| Subtask-Invarianten           | 38 (makeSubtaskAdd)|
| Completion-Workflow           | 38 (completeTodo)|
| 9 TODO-Tools                  | 38–40            |
| API-Coverage-Docs             | 11, 20, 27, 33, 41 |
| Infrastructure-home Commit    | 43–45            |

Alle Spec-Punkte abgedeckt.

**2. Placeholder scan:** Keine TBD/TODO im Plan; Verweise wie „copy lines 196-295 verbatim" sind explizite Anweisungen mit konkreter Zeilenspanne, kein Platzhalter.

**3. Type consistency:**

- `Pattern`-Struct hat `Type`, `Days`, `Day` — konsistent in `recurrence.go`, `recurrence_test.go`, `patternFromParams`.
- `IDParam` wird in `overtime.go` definiert und in allen anderen Dateien (`vacation.go`, `sickday.go`, `person.go`, `annual_event.go`, `project.go`, `todo.go`) genutzt.
- `resolveCompany`, `resolvePerson`, `resolveProject` Signaturen einheitlich: `(db *sql.DB, input string[, hint string]) (int64, string, error)`.
- `appLoc` in `time.go` definiert, genutzt in `time.go` und `annual_event.go`.

Keine Inkonsistenzen gefunden.

---

## Execution Handoff

Plan fertig und committet zu `docs/superpowers/plans/2026-04-18-personal-mcp-server.md`.

Zwei Ausführungsoptionen:

**1. Subagent-Driven (empfohlen)** — Frischer Subagent pro Task, Review zwischen Tasks, schnelle Iteration.

**2. Inline Execution** — Tasks in dieser Session ausführen mit Checkpoints.

Welchen Weg?
