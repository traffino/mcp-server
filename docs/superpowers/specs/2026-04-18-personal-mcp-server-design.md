# Personal MCP Server — Design

**Datum**: 2026-04-18
**Status**: Design

## Ziel

Der bestehende `tracker` MCP-Server wird zu einem umfassenderen Personal-Productivity-Hub ausgebaut und in `personal` umbenannt. Er vereint strukturierte, abfragbare persoenliche Daten: Arbeitszeit-Domaenen (Ueberstunden, Urlaub, Kranktage) bleiben per Company skopiert, dazu kommen Personen, jaehrliche Events, Projekte und eine dreistufige TODO-Hierarchie mit Rekurrenz.

Memory-Graph (`mcp-personal`) bleibt fuer Projekte, Organisationen, Techstack-Wissen und freien Personen-Kontext zustaendig; `personal` ergaenzt das mit Datums- und statusgetriebenen Queries.

## Scope

**Eingeschlossen**

- Rename `cmd/tracker/` → `cmd/personal/` inkl. Dockerfile, Makefile-Targets, README- und CLAUDE.md-Anpassungen
- Overtime-Schema erweitern um Start-/End-Zeit und zwei Entry-Typen (`work`, `reduction`)
- Neue Domaene `person` + `annual_event` (Birthday, Anniversary, Name Day)
- Neue Domaene `project`
- Neue Domaene `todo` mit Subtasks (strenge drei Ebenen) und wiederkehrenden TODOs (Wochentag- oder Monatstag-Pattern, fester Pattern-basierter Naechster-Termin)
- Timezone-Konfiguration via Env-Var, Default `Europe/Berlin`
- Infrastructure-home: Docker-Compose-Service-Rename
- Alter `cmd/tracker/`-Ordner wird im gleichen Schritt geloescht (keine Migration)

**Ausgeschlossen**

- Migration bestehender SQLite-Daten (Bestand ist vernachlaessigbar)
- Voll ausgebautes Kontakt-CRM mit Telefon/E-Mail/Adresse in `personal` (Memory bleibt primaere Person-Wissen-Quelle)
- Ueberstunden/Urlaub/Kranktage an Projekte binden — bleiben per Company
- Split-Shifts in einem Overtime-Eintrag

## Architektur-Ueberblick

Einzelnes Binary `personal`, `package main` in `cmd/personal/` mit mehreren Dateien (Monorepo-Konvention bleibt unveraendert). Keine neuen Shared-Packages unter `internal/`. Persistenz: SQLite ueber `modernc.org/sqlite` wie bisher, WAL + Foreign Keys.

### Datei-Layout `cmd/personal/`

```
main.go          Bootstrap, Tool-Registration, Shutdown
db.go            Schema, Migrationen (PRAGMA user_version)
helpers.go       errResult, textResult, nilIfEmpty, resolveCompany/Person/Project
time.go          Datum/Uhrzeit-Parsing, ComputeHours, Today() mit TZ
recurrence.go    Pattern-Parsing, NextOccurrence (pure Funktionen, getestet)
company.go       Company Tools
overtime.go      Overtime Tools (work + reduction)
vacation.go      Vacation Tools
sickday.go       Sick Day Tools
person.go        Person Tools
annual_event.go  Annual Event Tools
project.go       Project Tools
todo.go          TODO + Subtask Tools
```

### Konfiguration (Env-Vars)

| Variable           | Default             | Zweck                                                   |
|--------------------|---------------------|---------------------------------------------------------|
| `PERSONAL_DB_PATH` | `/data/personal.db` | SQLite-Datei                                            |
| `PERSONAL_TZ`      | `Europe/Berlin`     | IANA-Zeitzone fuer Today/Weekday/Recurrence-Berechnung  |
| `PORT`             | `:8000`             | MCP Streamable HTTP                                     |

`PERSONAL_TZ` wird beim Bootstrap via `time.LoadLocation` aufgeloest. Fehler bei unbekannter TZ ⇒ Fatal-Log mit Hinweis auf Default.

## Datenmodell

### Bestehend, unveraendert

```sql
CREATE TABLE company (
  id                   INTEGER PRIMARY KEY AUTOINCREMENT,
  name                 TEXT NOT NULL UNIQUE,
  weekly_hours         REAL,
  annual_vacation_days INTEGER,
  created_at           TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE vacation (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id INTEGER NOT NULL REFERENCES company(id) ON DELETE CASCADE,
  start_date TEXT NOT NULL,
  end_date   TEXT NOT NULL,
  days       REAL NOT NULL,
  type       TEXT NOT NULL DEFAULT 'vacation',
  note       TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_vacation_company_year ON vacation(company_id, start_date);

CREATE TABLE sick_day (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id INTEGER NOT NULL REFERENCES company(id) ON DELETE CASCADE,
  start_date TEXT NOT NULL,
  end_date   TEXT NOT NULL,
  days       INTEGER NOT NULL,
  note       TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_sick_day_company_year ON sick_day(company_id, start_date);
```

### Geaendert: `overtime`

```sql
CREATE TABLE overtime (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id INTEGER NOT NULL REFERENCES company(id) ON DELETE CASCADE,
  date       TEXT NOT NULL,                          -- YYYY-MM-DD
  type       TEXT NOT NULL CHECK(type IN ('work','reduction')),
  start_time TEXT,                                   -- HH:MM, bei type='work' Pflicht
  end_time   TEXT,                                   -- HH:MM, bei type='work' Pflicht
  hours      REAL NOT NULL CHECK(hours > 0),         -- bei work berechnet, bei reduction explizit (beides positiv)
  reason     TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  CHECK (
    (type='work'      AND start_time IS NOT NULL AND end_time IS NOT NULL) OR
    (type='reduction' AND start_time IS NULL     AND end_time IS NULL)
  )
);
CREATE INDEX idx_overtime_company_date ON overtime(company_id, date);
```

`hours` ist immer positiv gespeichert. Der Saldo diskriminiert per `type`:

```sql
SELECT SUM(CASE WHEN type='reduction' THEN -hours ELSE hours END) ...
```

### Neu: `person`

```sql
CREATE TABLE person (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  name       TEXT NOT NULL UNIQUE,
  note       TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
```

### Neu: `annual_event`

```sql
CREATE TABLE annual_event (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  person_id  INTEGER NOT NULL REFERENCES person(id) ON DELETE CASCADE,
  type       TEXT NOT NULL CHECK(type IN ('birthday','anniversary','name_day')),
  date       TEXT NOT NULL,                          -- YYYY-MM-DD, Jahr = erstes Auftreten (Altersberechnung)
  note       TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(person_id, type)
);
CREATE INDEX idx_annual_event_month_day ON annual_event(substr(date, 6));  -- "MM-DD" fuer Monatsfilter
```

### Neu: `project`

```sql
CREATE TABLE project (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  name       TEXT NOT NULL,
  company_id INTEGER REFERENCES company(id) ON DELETE SET NULL,
  status     TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','archived')),
  note       TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(company_id, name)
);
-- Private Projekte (company_id IS NULL) eindeutig nach Name (SQLite-NULL-Handling):
CREATE UNIQUE INDEX idx_project_private_name ON project(name) WHERE company_id IS NULL;
CREATE INDEX idx_project_company_status ON project(company_id, status);
```

### Neu: `todo`

```sql
CREATE TABLE todo (
  id                 INTEGER PRIMARY KEY AUTOINCREMENT,
  title              TEXT NOT NULL,
  project_id         INTEGER REFERENCES project(id) ON DELETE SET NULL,
  company_id         INTEGER REFERENCES company(id) ON DELETE SET NULL,
  parent_id          INTEGER REFERENCES todo(id) ON DELETE CASCADE,
  status             TEXT NOT NULL DEFAULT 'open'
                     CHECK(status IN ('open','in_progress','waiting','done','cancelled')),
  due_date           TEXT,
  note               TEXT,
  recurrence_pattern TEXT,                          -- JSON, nur bei Top-Level-TODOs
  completed_at       TEXT,
  created_at         TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_todo_status_due ON todo(status, due_date);
CREATE INDEX idx_todo_project ON todo(project_id);
CREATE INDEX idx_todo_parent ON todo(parent_id);
```

**Invarianten (applikativ durchgesetzt in Go)**:

- `parent_id IS NOT NULL` ⇒ Subtask. Subtasks haben `project_id = NULL` und `company_id = NULL` (erben vom Parent; bei Read werden Parent-Werte implizit mitgegeben).
- Parent eines Subtasks muss selbst `parent_id IS NULL` sein (strenge drei Ebenen).
- `recurrence_pattern` nur setzbar wenn `parent_id IS NULL`.
- Bei `status = 'done'`: `completed_at = datetime('now' in PERSONAL_TZ)`; bei Reset auf andere Status wird `completed_at` wieder `NULL`.

### Neu: `todo_completion`

```sql
CREATE TABLE todo_completion (
  id                     INTEGER PRIMARY KEY AUTOINCREMENT,
  todo_id                INTEGER NOT NULL REFERENCES todo(id) ON DELETE CASCADE,
  completed_at           TEXT NOT NULL,
  due_date_at_completion TEXT,
  created_at             TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_todo_completion_todo ON todo_completion(todo_id, completed_at);
```

Bei jedem Uebergang auf `status = 'done'` wird eine Zeile geschrieben (fuer einmalige wie wiederkehrende TODOs). Bei wiederkehrenden TODOs folgt direkt danach das Vorruecken von `due_date` (siehe Recurrence).

### Recurrence-Pattern

Strukturierter Param beim `todo_add`/`todo_update` (kein freies JSON in der MCP-UX). In der DB wird er als JSON serialisiert:

```json
{ "type": "weekday",  "days": [1,3,5] }    // ISO: Mo=1..So=7
{ "type": "monthday", "day": 15 }          // 1..31
```

**NextOccurrence-Semantik (fest, Pattern-basiert)**

- Nach Completion wird `due_date` auf die **naechste Pattern-Instanz strikt nach `max(today, current_due_date)`** gesetzt.
- Beispiel weekday=[1]: aktuelles `due_date` = `2026-04-13` (Mo), Done am `2026-04-15` (Mi) ⇒ neues `due_date` = `2026-04-20` (Mo).
- Beispiel monthday=15: aktuelles `due_date` = `2026-04-15`, Done am `2026-04-20` ⇒ neues `due_date` = `2026-05-15`.
- Monatstag > letzter Tag des Zielmonats (z.B. `day=31` im Februar) ⇒ **letzter Tag des Monats**.

### Migrations

SQLite `PRAGMA user_version`:

- `user_version = 0` ⇒ komplettes v1-Schema anwenden, danach `user_version = 1`.
- Spaetere Schemaaenderungen kommen als `migrateToV2`, etc., sequentiell.

Bestehende `tracker.db` wird durch Umbenennung/Neu-Deployment abgeloest (kein Migrationspfad).

## Tool-Katalog

Namenskonvention: `<domain>_<action>`. Fehlerausgaben englisch, Responses beschreibend mit IDs. Helfer `resolveCompany`, `resolvePerson`, `resolveProject` akzeptieren Name oder ID, case-insensitive; bei Mehrdeutigkeit Pflicht zur ID.

### Company (4)

- `company_create(name, weekly_hours?, annual_vacation_days?)`
- `company_list()`
- `company_update(id, name?, weekly_hours?, annual_vacation_days?)`
- `company_delete(id)` — Cascade

### Overtime (6)

- `overtime_add_work(company, date, start_time, end_time, reason?)` — `hours` aus `end − start` (Minuten-genau, 2 Nachkommastellen)
- `overtime_add_reduction(company, date, hours, reason?)` — `hours` positiv, wird im Saldo negativ gerechnet
- `overtime_list(company?, year?, month?, type?)`
- `overtime_update(id, date?, start_time?, end_time?, hours?, reason?)` — bei Zeit-Update `hours` automatisch neu berechnet; Typ-Wechsel nicht erlaubt
- `overtime_delete(id)`
- `overtime_summary(company?, year?, month?)` — Saldo mit `SUM(CASE WHEN type='reduction' THEN -hours ELSE hours END)`

### Vacation (5)

- `vacation_add`, `vacation_list`, `vacation_update`, `vacation_delete`, `vacation_balance`

### Sick Day (4)

- `sick_day_add`, `sick_day_list`, `sick_day_update`, `sick_day_delete`

### Person (4)

- `person_add(name, note?)`
- `person_list(search?)` — optional Name-Substring
- `person_update(id, name?, note?)`
- `person_delete(id)` — Cascade auf `annual_event`

### Annual Event (5)

- `annual_event_add(person, type, date, note?)`
- `annual_event_list(person?, type?, month?)` — Monat als 1..12
- `annual_event_update(id, date?, note?)`
- `annual_event_delete(id)`
- `annual_event_upcoming(days?)` — Default 30, sortiert nach naechstem Auftreten; Ausgabe mit Alter bzw. Jahrestag-Nummer

### Project (5)

- `project_add(name, company?, note?)`
- `project_list(company?, status?)` — `status` Default `active`
- `project_update(id, name?, company?, status?, note?)`
- `project_archive(id)` — Convenience-Alias
- `project_delete(id)` — Cascade auf TODOs (inkl. Subtasks); Erfolg nennt geloeschte Anzahl

### TODO + Subtask (9)

- `todo_add(title, project?, company?, due_date?, note?, recurrence_type?, recurrence_days?, recurrence_day?)` — Top-Level. `recurrence_type` ∈ `weekday`/`monthday`, dann entweder `recurrence_days` (Array 1..7) oder `recurrence_day` (1..31)
- `subtask_add(parent_id, title, due_date?, note?)` — Parent muss Top-Level sein
- `todo_list(project?, company?, status?, due_before?, due_after?, include_subtasks?)` — Default-Filter `status IN ('open','in_progress')`; bei explizit gesetztem `status` oder `status='all'` wird erweitert
- `todo_update(id, title?, due_date?, note?, status?, project?, company?, recurrence_type?, recurrence_days?, recurrence_day?, clear_recurrence?)` — Completion-Logik bei Wechsel auf `done`
- `todo_complete(id)` — Alias fuer `todo_update(id, status='done')`
- `todo_delete(id)` — Cascade auf Subtasks + Completion-History
- `todo_move(id, project?, company?)` — nur Top-Level
- `todo_upcoming(days?, company?)` — default 7 Tage, nur `open`/`in_progress`
- `todo_overdue(company?)` — `due_date < today()` mit Status `open`/`in_progress`

### Rendering Konvention `todo_list` mit `include_subtasks=true`

```
Todos:

  [12] Umzug vorbereiten | Haus | open | due 2026-05-01
    [13] Kisten besorgen | open
    [14] Moebelpacker anfragen | waiting
  [15] Steuererklaerung | Privat | open | due 2026-04-30 (recurring: monthday 30)
```

Einrueckung: zwei Spaces pro Ebene.

## Completion-Workflow (wiederkehrende TODOs)

Bei `todo_update(id, status='done')` mit nicht-leerem `recurrence_pattern`:

1. `INSERT INTO todo_completion (todo_id, completed_at, due_date_at_completion)`
2. `next = NextOccurrence(pattern, max(today, current due_date))`
3. `UPDATE todo SET due_date = next, status = 'open', completed_at = NULL`

Einmalige TODOs (kein `recurrence_pattern`):

1. `INSERT INTO todo_completion (...)`
2. `UPDATE todo SET status = 'done', completed_at = now()`

## Error Handling

Bestehende Konvention unveraendert (`errResult`/`textResult`). Neue Fehlermeldungen:

- `resolveProject` Ambiguitaet: `project %q is ambiguous — exists in companies X, Y. Use ID or pass company=<name>`
- `subtask_add` mit Subtask als Parent: `parent ID %d is already a subtask; subtasks cannot have children (max 3 levels)`
- `overtime_add_work` mit `end_time <= start_time`: `end_time must be after start_time`
- Ungueltiges Zeitformat: `start_time must be HH:MM`
- `todo_update` mit Rekurrenz-Param auf Subtask: `recurrence can only be set on top-level todos`
- Rekurrenz-Parameter-Konflikt: `recurrence_type='weekday' requires recurrence_days (1..7)`, `recurrence_type='monthday' requires recurrence_day (1..31)`
- `project_delete` mit Cascade: Erfolg enthaelt `Deleted project %q and N linked todos`

## Testing

Unit-Tests auf reine Funktionen:

- `recurrence.go`: `NextOccurrence` fuer Weekday- und Monthday-Pattern, inkl. Month-Overflow (31. im Feb), Sonntag-Wrap, Vergangenheits-Cursor
- `time.go`: `ComputeHours`, Datum-/Uhrzeit-Parser, `Today(loc)`-Abhaengigkeit

Keine DB-Integration-Tests im ersten Wurf (konsistent mit den anderen Servern im Monorepo).

Makefile-Target `make test` umfasst die bestehenden und neuen Tests.

## Delivery-Plan

Arbeit direkt auf `main`, logisch getrennte Commits:

1. **Rename + Docs**: `cmd/tracker/` → `cmd/personal/`, Dockerfile umbenannt, `Makefile`, `README.md`, `CLAUDE.md` angepasst. Keine Funktionsaenderung. Gruener Build.
2. **Overtime-Schema v2**: neue Spalten + CHECK, `overtime_add` aufgespalten in `overtime_add_work` + `overtime_add_reduction`, Summary-Query umgestellt.
3. **Person + Annual Event**: neue Tabellen + 9 Tools.
4. **Project**: neue Tabelle + 5 Tools.
5. **Recurrence + TODO**: `recurrence.go` inkl. Tests zuerst; dann `todo.go`, Tabellen + 9 Tools.

Nach allen Commits im `mcp-server`-Repo: separater Commit in `infrastructure-home` (Docker-Compose-Service `tracker` → `personal`, Image-Tag entsprechend). `PERSONAL_TZ=Europe/Berlin` als Default in Compose-Datei.

## Offene Punkte

Keine, sofern der Reviewer die Spec ohne Aenderungen abnimmt.
