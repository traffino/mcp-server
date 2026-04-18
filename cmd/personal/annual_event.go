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
