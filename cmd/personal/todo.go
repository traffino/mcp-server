package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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
	if _, err := ParsePattern(serialized); err != nil {
		return Pattern{}, false, err
	}
	return p, true, nil
}

type TodoAddParams struct {
	Title          string `json:"title" jsonschema:"TODO title"`
	Project        string `json:"project,omitempty" jsonschema:"Project name or ID"`
	Company        string `json:"company,omitempty" jsonschema:"Company name or ID (if no project)"`
	DueDate        string `json:"due_date,omitempty" jsonschema:"Due date YYYY-MM-DD"`
	Note           string `json:"note,omitempty" jsonschema:"Optional note"`
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
			id      int64
			title   string
			project string
			company string
			status  string
			due     string
			pattern sql.NullString
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
			if p.Status == "done" {
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
