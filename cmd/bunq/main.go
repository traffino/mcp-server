package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/traffino/mcp-server/internal/config"
	"github.com/traffino/mcp-server/internal/server"
)

const baseURL = "https://api.bunq.com/v1"

var httpClient = &http.Client{Timeout: 30 * time.Second}

func main() {
	apiKey := config.Require("BUNQ_API_KEY")
	srv := server.New("bunq", "1.0.0")
	s := srv.MCPServer()

	// Accounts
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_accounts",
		Description: "List all monetary accounts with balances",
	}, makeListAccounts(apiKey))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_account",
		Description: "Get details of a specific monetary account",
	}, makeGetAccount(apiKey))

	// Payments
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_payments",
		Description: "List payments for a monetary account",
	}, makeListPayments(apiKey))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_payment",
		Description: "Get details of a specific payment",
	}, makeGetPayment(apiKey))

	// Cards
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_cards",
		Description: "List all cards",
	}, makeListCards(apiKey))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_card",
		Description: "Get details of a specific card",
	}, makeGetCard(apiKey))

	// Schedules
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_schedules",
		Description: "List scheduled payments for an account",
	}, makeListSchedules(apiKey))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_schedule",
		Description: "Get details of a specific scheduled payment",
	}, makeGetSchedule(apiKey))

	srv.ListenAndServe(config.Get("PORT", ":8000"))
}

// --- Param Types ---

type UserIDParams struct {
	UserID int64 `json:"user_id" jsonschema:"bunq user ID"`
}

type AccountParams struct {
	UserID    int64 `json:"user_id" jsonschema:"bunq user ID"`
	AccountID int64 `json:"account_id" jsonschema:"Monetary account ID"`
}

type PaymentListParams struct {
	UserID    int64 `json:"user_id" jsonschema:"bunq user ID"`
	AccountID int64 `json:"account_id" jsonschema:"Monetary account ID"`
	Count     int   `json:"count,omitempty" jsonschema:"Number of payments to return (default 10)"`
}

type PaymentParams struct {
	UserID    int64 `json:"user_id" jsonschema:"bunq user ID"`
	AccountID int64 `json:"account_id" jsonschema:"Monetary account ID"`
	PaymentID int64 `json:"payment_id" jsonschema:"Payment ID"`
}

type CardParams struct {
	UserID int64 `json:"user_id" jsonschema:"bunq user ID"`
	CardID int64 `json:"card_id" jsonschema:"Card ID"`
}

type ScheduleParams struct {
	UserID     int64 `json:"user_id" jsonschema:"bunq user ID"`
	AccountID  int64 `json:"account_id" jsonschema:"Monetary account ID"`
	ScheduleID int64 `json:"schedule_id" jsonschema:"Schedule ID"`
}

// --- Handlers ---

func makeListAccounts(token string) func(context.Context, *mcp.CallToolRequest, *UserIDParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *UserIDParams) (*mcp.CallToolResult, any, error) {
		if p.UserID <= 0 {
			return errResult("user_id is required")
		}
		return bunqGet(token, fmt.Sprintf("/user/%d/monetary-account", p.UserID))
	}
}

func makeGetAccount(token string) func(context.Context, *mcp.CallToolRequest, *AccountParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *AccountParams) (*mcp.CallToolResult, any, error) {
		if p.UserID <= 0 || p.AccountID <= 0 {
			return errResult("user_id and account_id are required")
		}
		return bunqGet(token, fmt.Sprintf("/user/%d/monetary-account/%d", p.UserID, p.AccountID))
	}
}

func makeListPayments(token string) func(context.Context, *mcp.CallToolRequest, *PaymentListParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *PaymentListParams) (*mcp.CallToolResult, any, error) {
		if p.UserID <= 0 || p.AccountID <= 0 {
			return errResult("user_id and account_id are required")
		}
		path := fmt.Sprintf("/user/%d/monetary-account/%d/payment", p.UserID, p.AccountID)
		if p.Count > 0 {
			path += fmt.Sprintf("?count=%d", p.Count)
		}
		return bunqGet(token, path)
	}
}

func makeGetPayment(token string) func(context.Context, *mcp.CallToolRequest, *PaymentParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *PaymentParams) (*mcp.CallToolResult, any, error) {
		if p.UserID <= 0 || p.AccountID <= 0 || p.PaymentID <= 0 {
			return errResult("user_id, account_id, and payment_id are required")
		}
		return bunqGet(token, fmt.Sprintf("/user/%d/monetary-account/%d/payment/%d", p.UserID, p.AccountID, p.PaymentID))
	}
}

func makeListCards(token string) func(context.Context, *mcp.CallToolRequest, *UserIDParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *UserIDParams) (*mcp.CallToolResult, any, error) {
		if p.UserID <= 0 {
			return errResult("user_id is required")
		}
		return bunqGet(token, fmt.Sprintf("/user/%d/card", p.UserID))
	}
}

func makeGetCard(token string) func(context.Context, *mcp.CallToolRequest, *CardParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *CardParams) (*mcp.CallToolResult, any, error) {
		if p.UserID <= 0 || p.CardID <= 0 {
			return errResult("user_id and card_id are required")
		}
		return bunqGet(token, fmt.Sprintf("/user/%d/card/%d", p.UserID, p.CardID))
	}
}

func makeListSchedules(token string) func(context.Context, *mcp.CallToolRequest, *AccountParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *AccountParams) (*mcp.CallToolResult, any, error) {
		if p.UserID <= 0 || p.AccountID <= 0 {
			return errResult("user_id and account_id are required")
		}
		return bunqGet(token, fmt.Sprintf("/user/%d/monetary-account/%d/schedule", p.UserID, p.AccountID))
	}
}

func makeGetSchedule(token string) func(context.Context, *mcp.CallToolRequest, *ScheduleParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ScheduleParams) (*mcp.CallToolResult, any, error) {
		if p.UserID <= 0 || p.AccountID <= 0 || p.ScheduleID <= 0 {
			return errResult("user_id, account_id, and schedule_id are required")
		}
		return bunqGet(token, fmt.Sprintf("/user/%d/monetary-account/%d/schedule/%d", p.UserID, p.AccountID, p.ScheduleID))
	}
}

// --- API Client ---

func bunqGet(token, path string) (*mcp.CallToolResult, any, error) {
	req, err := http.NewRequest("GET", baseURL+path, nil)
	if err != nil {
		return errResult(err.Error())
	}

	req.Header.Set("X-Bunq-Client-Authentication", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "traffino-mcp-bunq/1.0.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return errResult(err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errResult(err.Error())
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errResult(fmt.Sprintf("bunq API error %d: %s", resp.StatusCode, string(body)))
	}

	var pretty json.RawMessage
	if json.Unmarshal(body, &pretty) == nil {
		if indented, err := json.MarshalIndent(pretty, "", "  "); err == nil {
			body = indented
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(body)}},
	}, nil, nil
}

func errResult(msg string) (*mcp.CallToolResult, any, error) {
	r := &mcp.CallToolResult{}
	r.SetError(fmt.Errorf("%s", msg))
	return r, nil, nil
}
