package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/traffino/mcp-server/internal/config"
	"github.com/traffino/mcp-server/internal/server"
)

const graphURL = "https://graph.microsoft.com/v1.0"
const tokenURL = "https://login.microsoftonline.com/%s/oauth2/v2.0/token"

var httpClient = &http.Client{Timeout: 30 * time.Second}

func main() {
	clientID := config.Require("MS365_MCP_CLIENT_ID")
	clientSecret := config.Require("MS365_MCP_CLIENT_SECRET")
	tenantID := config.Require("MS365_MCP_TENANT_ID")

	auth := &tokenProvider{
		clientID: clientID, clientSecret: clientSecret, tenantID: tenantID,
	}

	srv := server.New("ms365", "1.0.0")
	s := srv.MCPServer()

	// Mail
	mcp.AddTool(s, &mcp.Tool{Name: "list_messages", Description: "List email messages (supports shared mailbox via user param)"}, makeListMessages(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "get_message", Description: "Get email message details"}, makeGetMessage(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "send_mail", Description: "Send an email"}, makeSendMail(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "reply_to_message", Description: "Reply to an email"}, makeReplyMessage(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "forward_message", Description: "Forward an email"}, makeForwardMessage(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "move_message", Description: "Move a message to a folder"}, makeMoveMessage(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "delete_message", Description: "Delete a message"}, makeDeleteMessage(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "mark_read", Description: "Mark a message as read or unread"}, makeMarkRead(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "list_mail_folders", Description: "List mail folders"}, makeListMailFolders(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "search_messages", Description: "Search messages"}, makeSearchMessages(auth))

	// Calendar
	mcp.AddTool(s, &mcp.Tool{Name: "list_events", Description: "List calendar events"}, makeListEvents(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "get_event", Description: "Get calendar event details"}, makeGetEvent(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "create_event", Description: "Create a calendar event"}, makeCreateEvent(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "update_event", Description: "Update a calendar event"}, makeUpdateEvent(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "delete_event", Description: "Delete a calendar event"}, makeDeleteEvent(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "respond_to_event", Description: "Accept, tentatively accept, or decline an event"}, makeRespondEvent(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "list_calendars", Description: "List all calendars"}, makeUserGet(auth, "/calendars"))

	// Contacts
	mcp.AddTool(s, &mcp.Tool{Name: "list_contacts", Description: "List contacts"}, makeUserGet(auth, "/contacts"))
	mcp.AddTool(s, &mcp.Tool{Name: "get_contact", Description: "Get contact details"}, makeGetByID(auth, "/contacts"))
	mcp.AddTool(s, &mcp.Tool{Name: "create_contact", Description: "Create a contact"}, makeCreateContact(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "delete_contact", Description: "Delete a contact"}, makeDeleteByID(auth, "/contacts"))
	mcp.AddTool(s, &mcp.Tool{Name: "search_contacts", Description: "Search contacts by name or email"}, makeSearchContacts(auth))

	// OneDrive
	mcp.AddTool(s, &mcp.Tool{Name: "list_drive_items", Description: "List files and folders in OneDrive"}, makeListDriveItems(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "get_drive_item", Description: "Get file/folder details"}, makeGetDriveItem(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "search_drive", Description: "Search files in OneDrive"}, makeSearchDrive(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "create_folder", Description: "Create a folder in OneDrive"}, makeCreateFolder(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "delete_drive_item", Description: "Delete a file or folder"}, makeDeleteDriveItem(auth))

	// Teams
	mcp.AddTool(s, &mcp.Tool{Name: "list_teams", Description: "List joined teams"}, makeUserGet(auth, "/joinedTeams"))
	mcp.AddTool(s, &mcp.Tool{Name: "list_team_channels", Description: "List channels in a team"}, makeListTeamChannels(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "list_channel_messages", Description: "List messages in a channel"}, makeListChannelMessages(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "send_channel_message", Description: "Send a message to a channel"}, makeSendChannelMessage(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "create_channel", Description: "Create a channel in a team"}, makeCreateChannel(auth))

	// OneNote
	mcp.AddTool(s, &mcp.Tool{Name: "list_notebooks", Description: "List OneNote notebooks"}, makeUserGet(auth, "/onenote/notebooks"))
	mcp.AddTool(s, &mcp.Tool{Name: "list_notebook_pages", Description: "List pages in a notebook section"}, makeListNotebookPages(auth))

	// To Do
	mcp.AddTool(s, &mcp.Tool{Name: "list_todo_lists", Description: "List To Do task lists"}, makeUserGet(auth, "/todo/lists"))
	mcp.AddTool(s, &mcp.Tool{Name: "list_tasks", Description: "List tasks in a To Do list"}, makeListTasks(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "create_task", Description: "Create a task in a To Do list"}, makeCreateTask(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "update_task", Description: "Update a task"}, makeUpdateTask(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "complete_task", Description: "Mark a task as completed"}, makeCompleteTask(auth))
	mcp.AddTool(s, &mcp.Tool{Name: "delete_task", Description: "Delete a task"}, makeDeleteTask(auth))

	// User/Profile
	mcp.AddTool(s, &mcp.Tool{Name: "get_profile", Description: "Get current user profile"}, makeUserGet(auth, ""))

	srv.ListenAndServe(config.Get("PORT", ":8000"))
}

// --- OAuth Token Provider ---

type tokenProvider struct {
	clientID, clientSecret, tenantID string
	mu                               sync.Mutex
	token                            string
	expiry                           time.Time
}

func (tp *tokenProvider) getToken() (string, error) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	if tp.token != "" && time.Now().Before(tp.expiry) {
		return tp.token, nil
	}

	data := fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s&scope=https://graph.microsoft.com/.default",
		tp.clientID, tp.clientSecret)

	resp, err := http.Post(
		fmt.Sprintf(tokenURL, tp.tenantID),
		"application/x-www-form-urlencoded",
		bytes.NewBufferString(data),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", fmt.Errorf("OAuth error: %s - %s", result.Error, result.ErrorDesc)
	}

	tp.token = result.AccessToken
	tp.expiry = time.Now().Add(time.Duration(result.ExpiresIn-60) * time.Second)
	return tp.token, nil
}

// --- Param Types ---

type UserParam struct {
	User string `json:"user,omitempty" jsonschema:"User principal name or ID for shared mailbox (default: me)"`
}

type MessageListParams struct {
	User    string `json:"user,omitempty" jsonschema:"User for shared mailbox (default: me)"`
	Folder  string `json:"folder,omitempty" jsonschema:"Mail folder ID (default: inbox)"`
	Top     int    `json:"top,omitempty" jsonschema:"Number of messages (default 10)"`
	Filter  string `json:"filter,omitempty" jsonschema:"OData filter expression"`
	OrderBy string `json:"orderby,omitempty" jsonschema:"Sort, e.g. receivedDateTime desc"`
}

type MessageIDParam struct {
	User      string `json:"user,omitempty" jsonschema:"User for shared mailbox (default: me)"`
	MessageID string `json:"message_id" jsonschema:"Message ID"`
}

type SendMailParams struct {
	User    string   `json:"user,omitempty" jsonschema:"Send as user (default: me)"`
	To      []string `json:"to" jsonschema:"Recipient email addresses"`
	Subject string   `json:"subject" jsonschema:"Email subject"`
	Body    string   `json:"body" jsonschema:"Email body (HTML supported)"`
	CC      []string `json:"cc,omitempty" jsonschema:"CC recipients"`
}

type ReplyParams struct {
	User      string `json:"user,omitempty" jsonschema:"User (default: me)"`
	MessageID string `json:"message_id" jsonschema:"Message ID to reply to"`
	Comment   string `json:"comment" jsonschema:"Reply text"`
}

type ForwardParams struct {
	User      string   `json:"user,omitempty" jsonschema:"User (default: me)"`
	MessageID string   `json:"message_id" jsonschema:"Message ID to forward"`
	To        []string `json:"to" jsonschema:"Forward to emails"`
	Comment   string   `json:"comment,omitempty" jsonschema:"Comment to include"`
}

type MoveParams struct {
	User        string `json:"user,omitempty" jsonschema:"User (default: me)"`
	MessageID   string `json:"message_id" jsonschema:"Message ID"`
	Destination string `json:"destination" jsonschema:"Destination folder ID"`
}

type MarkReadParams struct {
	User      string `json:"user,omitempty" jsonschema:"User (default: me)"`
	MessageID string `json:"message_id" jsonschema:"Message ID"`
	IsRead    bool   `json:"is_read" jsonschema:"true to mark read, false for unread"`
}

type SearchMessagesParams struct {
	User  string `json:"user,omitempty" jsonschema:"User (default: me)"`
	Query string `json:"query" jsonschema:"Search query"`
	Top   int    `json:"top,omitempty" jsonschema:"Number of results (default 10)"`
}

type EventListParams struct {
	User  string `json:"user,omitempty" jsonschema:"User (default: me)"`
	Top   int    `json:"top,omitempty" jsonschema:"Number of events"`
	Start string `json:"start,omitempty" jsonschema:"Start datetime ISO 8601"`
	End   string `json:"end,omitempty" jsonschema:"End datetime ISO 8601"`
}

type EventIDParam struct {
	User    string `json:"user,omitempty" jsonschema:"User (default: me)"`
	EventID string `json:"event_id" jsonschema:"Event ID"`
}

type CreateEventParams struct {
	User      string   `json:"user,omitempty" jsonschema:"User (default: me)"`
	Subject   string   `json:"subject" jsonschema:"Event subject"`
	Start     string   `json:"start" jsonschema:"Start datetime ISO 8601"`
	End       string   `json:"end" jsonschema:"End datetime ISO 8601"`
	TimeZone  string   `json:"timezone,omitempty" jsonschema:"Time zone (default UTC)"`
	Body      string   `json:"body,omitempty" jsonschema:"Event body"`
	Location  string   `json:"location,omitempty" jsonschema:"Location"`
	Attendees []string `json:"attendees,omitempty" jsonschema:"Attendee email addresses"`
}

type RespondEventParams struct {
	User     string `json:"user,omitempty" jsonschema:"User (default: me)"`
	EventID  string `json:"event_id" jsonschema:"Event ID"`
	Response string `json:"response" jsonschema:"Response: accept, tentativelyAccept, decline"`
	Comment  string `json:"comment,omitempty" jsonschema:"Response comment"`
}

type CreateContactParams struct {
	User        string `json:"user,omitempty" jsonschema:"User (default: me)"`
	GivenName   string `json:"given_name" jsonschema:"First name"`
	Surname     string `json:"surname,omitempty" jsonschema:"Last name"`
	Email       string `json:"email,omitempty" jsonschema:"Email address"`
	Phone       string `json:"phone,omitempty" jsonschema:"Phone number"`
	CompanyName string `json:"company_name,omitempty" jsonschema:"Company"`
}

type SearchContactsParams struct {
	User  string `json:"user,omitempty" jsonschema:"User (default: me)"`
	Query string `json:"query" jsonschema:"Search query (name or email)"`
}

type DriveListParams struct {
	User   string `json:"user,omitempty" jsonschema:"User (default: me)"`
	Path   string `json:"path,omitempty" jsonschema:"Folder path (default: root)"`
	ItemID string `json:"item_id,omitempty" jsonschema:"Item ID (alternative to path)"`
}

type DriveItemParam struct {
	User   string `json:"user,omitempty" jsonschema:"User (default: me)"`
	ItemID string `json:"item_id" jsonschema:"Drive item ID"`
}

type SearchDriveParams struct {
	User  string `json:"user,omitempty" jsonschema:"User (default: me)"`
	Query string `json:"query" jsonschema:"Search query"`
}

type CreateFolderParams struct {
	User     string `json:"user,omitempty" jsonschema:"User (default: me)"`
	ParentID string `json:"parent_id,omitempty" jsonschema:"Parent folder ID (default: root)"`
	Name     string `json:"name" jsonschema:"Folder name"`
}

type TeamIDParam struct {
	TeamID string `json:"team_id" jsonschema:"Team ID"`
}

type ChannelMsgParams struct {
	TeamID    string `json:"team_id" jsonschema:"Team ID"`
	ChannelID string `json:"channel_id" jsonschema:"Channel ID"`
	Top       int    `json:"top,omitempty" jsonschema:"Number of messages"`
}

type SendChannelMsgParams struct {
	TeamID    string `json:"team_id" jsonschema:"Team ID"`
	ChannelID string `json:"channel_id" jsonschema:"Channel ID"`
	Content   string `json:"content" jsonschema:"Message content (HTML supported)"`
}

type CreateChannelParams struct {
	TeamID      string `json:"team_id" jsonschema:"Team ID"`
	DisplayName string `json:"display_name" jsonschema:"Channel name"`
	Description string `json:"description,omitempty" jsonschema:"Channel description"`
}

type SectionIDParam struct {
	User      string `json:"user,omitempty" jsonschema:"User (default: me)"`
	SectionID string `json:"section_id" jsonschema:"OneNote section ID"`
}

type TodoListIDParam struct {
	User   string `json:"user,omitempty" jsonschema:"User (default: me)"`
	ListID string `json:"list_id" jsonschema:"To Do list ID"`
}

type CreateTaskParams struct {
	User    string `json:"user,omitempty" jsonschema:"User (default: me)"`
	ListID  string `json:"list_id" jsonschema:"To Do list ID"`
	Title   string `json:"title" jsonschema:"Task title"`
	DueDate string `json:"due_date,omitempty" jsonschema:"Due date YYYY-MM-DD"`
	Body    string `json:"body,omitempty" jsonschema:"Task body"`
}

type TaskIDParam struct {
	User   string `json:"user,omitempty" jsonschema:"User (default: me)"`
	ListID string `json:"list_id" jsonschema:"To Do list ID"`
	TaskID string `json:"task_id" jsonschema:"Task ID"`
}

type IDParam struct {
	User string `json:"user,omitempty" jsonschema:"User (default: me)"`
	ID   string `json:"id" jsonschema:"Resource ID"`
}

// --- Helper: user path prefix ---

func userPrefix(user string) string {
	if user != "" {
		return fmt.Sprintf("/users/%s", user)
	}
	return "/me"
}

// --- Generic Handlers ---

func makeUserGet(auth *tokenProvider, suffix string) func(context.Context, *mcp.CallToolRequest, *UserParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *UserParam) (*mcp.CallToolResult, any, error) {
		return graphGet(auth, userPrefix(p.User)+suffix)
	}
}

func makeGetByID(auth *tokenProvider, resource string) func(context.Context, *mcp.CallToolRequest, *IDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IDParam) (*mcp.CallToolResult, any, error) {
		if p.ID == "" {
			return errResult("id is required")
		}
		return graphGet(auth, fmt.Sprintf("%s%s/%s", userPrefix(p.User), resource, p.ID))
	}
}

func makeDeleteByID(auth *tokenProvider, resource string) func(context.Context, *mcp.CallToolRequest, *IDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IDParam) (*mcp.CallToolResult, any, error) {
		if p.ID == "" {
			return errResult("id is required")
		}
		return graphDelete(auth, fmt.Sprintf("%s%s/%s", userPrefix(p.User), resource, p.ID))
	}
}

// --- Mail Handlers ---

func makeListMessages(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *MessageListParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *MessageListParams) (*mcp.CallToolResult, any, error) {
		folder := "inbox"
		if p.Folder != "" {
			folder = p.Folder
		}
		path := fmt.Sprintf("%s/mailFolders/%s/messages", userPrefix(p.User), folder)
		qp := "?"
		if p.Top > 0 {
			qp += fmt.Sprintf("$top=%d&", p.Top)
		}
		if p.Filter != "" {
			qp += fmt.Sprintf("$filter=%s&", p.Filter)
		}
		if p.OrderBy != "" {
			qp += fmt.Sprintf("$orderby=%s&", p.OrderBy)
		}
		return graphGet(auth, path+qp)
	}
}

func makeGetMessage(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *MessageIDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *MessageIDParam) (*mcp.CallToolResult, any, error) {
		if p.MessageID == "" {
			return errResult("message_id is required")
		}
		return graphGet(auth, fmt.Sprintf("%s/messages/%s", userPrefix(p.User), p.MessageID))
	}
}

func makeSendMail(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *SendMailParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *SendMailParams) (*mcp.CallToolResult, any, error) {
		if len(p.To) == 0 || p.Subject == "" {
			return errResult("to and subject are required")
		}
		toRecipients := make([]map[string]any, len(p.To))
		for i, email := range p.To {
			toRecipients[i] = map[string]any{"emailAddress": map[string]any{"address": email}}
		}
		message := map[string]any{
			"subject":      p.Subject,
			"body":         map[string]any{"contentType": "HTML", "content": p.Body},
			"toRecipients": toRecipients,
		}
		if len(p.CC) > 0 {
			ccRecipients := make([]map[string]any, len(p.CC))
			for i, email := range p.CC {
				ccRecipients[i] = map[string]any{"emailAddress": map[string]any{"address": email}}
			}
			message["ccRecipients"] = ccRecipients
		}
		body := map[string]any{"message": message, "saveToSentItems": true}
		return graphPost(auth, userPrefix(p.User)+"/sendMail", body)
	}
}

func makeReplyMessage(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *ReplyParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ReplyParams) (*mcp.CallToolResult, any, error) {
		if p.MessageID == "" || p.Comment == "" {
			return errResult("message_id and comment are required")
		}
		body := map[string]any{"comment": p.Comment}
		return graphPost(auth, fmt.Sprintf("%s/messages/%s/reply", userPrefix(p.User), p.MessageID), body)
	}
}

func makeForwardMessage(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *ForwardParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ForwardParams) (*mcp.CallToolResult, any, error) {
		if p.MessageID == "" || len(p.To) == 0 {
			return errResult("message_id and to are required")
		}
		toRecipients := make([]map[string]any, len(p.To))
		for i, email := range p.To {
			toRecipients[i] = map[string]any{"emailAddress": map[string]any{"address": email}}
		}
		body := map[string]any{"toRecipients": toRecipients}
		if p.Comment != "" {
			body["comment"] = p.Comment
		}
		return graphPost(auth, fmt.Sprintf("%s/messages/%s/forward", userPrefix(p.User), p.MessageID), body)
	}
}

func makeMoveMessage(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *MoveParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *MoveParams) (*mcp.CallToolResult, any, error) {
		if p.MessageID == "" || p.Destination == "" {
			return errResult("message_id and destination are required")
		}
		body := map[string]any{"destinationId": p.Destination}
		return graphPost(auth, fmt.Sprintf("%s/messages/%s/move", userPrefix(p.User), p.MessageID), body)
	}
}

func makeDeleteMessage(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *MessageIDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *MessageIDParam) (*mcp.CallToolResult, any, error) {
		if p.MessageID == "" {
			return errResult("message_id is required")
		}
		return graphDelete(auth, fmt.Sprintf("%s/messages/%s", userPrefix(p.User), p.MessageID))
	}
}

func makeMarkRead(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *MarkReadParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *MarkReadParams) (*mcp.CallToolResult, any, error) {
		if p.MessageID == "" {
			return errResult("message_id is required")
		}
		body := map[string]any{"isRead": p.IsRead}
		return graphPatch(auth, fmt.Sprintf("%s/messages/%s", userPrefix(p.User), p.MessageID), body)
	}
}

func makeListMailFolders(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *UserParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *UserParam) (*mcp.CallToolResult, any, error) {
		return graphGet(auth, userPrefix(p.User)+"/mailFolders")
	}
}

func makeSearchMessages(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *SearchMessagesParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *SearchMessagesParams) (*mcp.CallToolResult, any, error) {
		if p.Query == "" {
			return errResult("query is required")
		}
		path := fmt.Sprintf("%s/messages?$search=\"%s\"", userPrefix(p.User), p.Query)
		if p.Top > 0 {
			path += fmt.Sprintf("&$top=%d", p.Top)
		}
		return graphGet(auth, path)
	}
}

// --- Calendar Handlers ---

func makeListEvents(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *EventListParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *EventListParams) (*mcp.CallToolResult, any, error) {
		path := userPrefix(p.User)
		if p.Start != "" && p.End != "" {
			path += fmt.Sprintf("/calendarView?startDateTime=%s&endDateTime=%s", p.Start, p.End)
		} else {
			path += "/events?"
		}
		if p.Top > 0 {
			path += fmt.Sprintf("&$top=%d", p.Top)
		}
		return graphGet(auth, path)
	}
}

func makeGetEvent(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *EventIDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *EventIDParam) (*mcp.CallToolResult, any, error) {
		if p.EventID == "" {
			return errResult("event_id is required")
		}
		return graphGet(auth, fmt.Sprintf("%s/events/%s", userPrefix(p.User), p.EventID))
	}
}

func makeCreateEvent(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *CreateEventParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *CreateEventParams) (*mcp.CallToolResult, any, error) {
		if p.Subject == "" || p.Start == "" || p.End == "" {
			return errResult("subject, start, and end are required")
		}
		tz := "UTC"
		if p.TimeZone != "" {
			tz = p.TimeZone
		}
		event := map[string]any{
			"subject": p.Subject,
			"start":   map[string]any{"dateTime": p.Start, "timeZone": tz},
			"end":     map[string]any{"dateTime": p.End, "timeZone": tz},
		}
		if p.Body != "" {
			event["body"] = map[string]any{"contentType": "HTML", "content": p.Body}
		}
		if p.Location != "" {
			event["location"] = map[string]any{"displayName": p.Location}
		}
		if len(p.Attendees) > 0 {
			attendees := make([]map[string]any, len(p.Attendees))
			for i, email := range p.Attendees {
				attendees[i] = map[string]any{
					"emailAddress": map[string]any{"address": email},
					"type":         "required",
				}
			}
			event["attendees"] = attendees
		}
		return graphPost(auth, userPrefix(p.User)+"/events", event)
	}
}

func makeUpdateEvent(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *CreateEventParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *CreateEventParams) (*mcp.CallToolResult, any, error) {
		// Reuse CreateEventParams but require event_id via subject workaround — actually we need a separate type
		return errResult("use event_id-based update (not yet exposed separately)")
	}
}

func makeDeleteEvent(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *EventIDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *EventIDParam) (*mcp.CallToolResult, any, error) {
		if p.EventID == "" {
			return errResult("event_id is required")
		}
		return graphDelete(auth, fmt.Sprintf("%s/events/%s", userPrefix(p.User), p.EventID))
	}
}

func makeRespondEvent(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *RespondEventParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *RespondEventParams) (*mcp.CallToolResult, any, error) {
		if p.EventID == "" || p.Response == "" {
			return errResult("event_id and response are required")
		}
		body := map[string]any{"sendResponse": true}
		if p.Comment != "" {
			body["comment"] = p.Comment
		}
		return graphPost(auth, fmt.Sprintf("%s/events/%s/%s", userPrefix(p.User), p.EventID, p.Response), body)
	}
}

// --- Contact Handlers ---

func makeCreateContact(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *CreateContactParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *CreateContactParams) (*mcp.CallToolResult, any, error) {
		if p.GivenName == "" {
			return errResult("given_name is required")
		}
		contact := map[string]any{"givenName": p.GivenName}
		if p.Surname != "" {
			contact["surname"] = p.Surname
		}
		if p.Email != "" {
			contact["emailAddresses"] = []map[string]any{{"address": p.Email}}
		}
		if p.Phone != "" {
			contact["businessPhones"] = []string{p.Phone}
		}
		if p.CompanyName != "" {
			contact["companyName"] = p.CompanyName
		}
		return graphPost(auth, userPrefix(p.User)+"/contacts", contact)
	}
}

func makeSearchContacts(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *SearchContactsParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *SearchContactsParams) (*mcp.CallToolResult, any, error) {
		if p.Query == "" {
			return errResult("query is required")
		}
		path := fmt.Sprintf("%s/contacts?$filter=contains(displayName,'%s') or contains(emailAddresses/any(e:e/address),'%s')",
			userPrefix(p.User), p.Query, p.Query)
		return graphGet(auth, path)
	}
}

// --- OneDrive Handlers ---

func makeListDriveItems(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *DriveListParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *DriveListParams) (*mcp.CallToolResult, any, error) {
		prefix := userPrefix(p.User)
		if p.ItemID != "" {
			return graphGet(auth, fmt.Sprintf("%s/drive/items/%s/children", prefix, p.ItemID))
		}
		if p.Path != "" && p.Path != "/" {
			return graphGet(auth, fmt.Sprintf("%s/drive/root:/%s:/children", prefix, p.Path))
		}
		return graphGet(auth, prefix+"/drive/root/children")
	}
}

func makeGetDriveItem(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *DriveItemParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *DriveItemParam) (*mcp.CallToolResult, any, error) {
		if p.ItemID == "" {
			return errResult("item_id is required")
		}
		return graphGet(auth, fmt.Sprintf("%s/drive/items/%s", userPrefix(p.User), p.ItemID))
	}
}

func makeSearchDrive(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *SearchDriveParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *SearchDriveParams) (*mcp.CallToolResult, any, error) {
		if p.Query == "" {
			return errResult("query is required")
		}
		return graphGet(auth, fmt.Sprintf("%s/drive/root/search(q='%s')", userPrefix(p.User), p.Query))
	}
}

func makeCreateFolder(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *CreateFolderParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *CreateFolderParams) (*mcp.CallToolResult, any, error) {
		if p.Name == "" {
			return errResult("name is required")
		}
		parent := "root"
		if p.ParentID != "" {
			parent = fmt.Sprintf("items/%s", p.ParentID)
		}
		body := map[string]any{
			"name":                              p.Name,
			"folder":                            map[string]any{},
			"@microsoft.graph.conflictBehavior": "rename",
		}
		return graphPost(auth, fmt.Sprintf("%s/drive/%s/children", userPrefix(p.User), parent), body)
	}
}

func makeDeleteDriveItem(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *DriveItemParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *DriveItemParam) (*mcp.CallToolResult, any, error) {
		if p.ItemID == "" {
			return errResult("item_id is required")
		}
		return graphDelete(auth, fmt.Sprintf("%s/drive/items/%s", userPrefix(p.User), p.ItemID))
	}
}

// --- Teams Handlers ---

func makeListTeamChannels(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *TeamIDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *TeamIDParam) (*mcp.CallToolResult, any, error) {
		if p.TeamID == "" {
			return errResult("team_id is required")
		}
		return graphGet(auth, fmt.Sprintf("/teams/%s/channels", p.TeamID))
	}
}

func makeListChannelMessages(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *ChannelMsgParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ChannelMsgParams) (*mcp.CallToolResult, any, error) {
		if p.TeamID == "" || p.ChannelID == "" {
			return errResult("team_id and channel_id are required")
		}
		path := fmt.Sprintf("/teams/%s/channels/%s/messages", p.TeamID, p.ChannelID)
		if p.Top > 0 {
			path += fmt.Sprintf("?$top=%d", p.Top)
		}
		return graphGet(auth, path)
	}
}

func makeSendChannelMessage(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *SendChannelMsgParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *SendChannelMsgParams) (*mcp.CallToolResult, any, error) {
		if p.TeamID == "" || p.ChannelID == "" || p.Content == "" {
			return errResult("team_id, channel_id, and content are required")
		}
		body := map[string]any{
			"body": map[string]any{"contentType": "html", "content": p.Content},
		}
		return graphPost(auth, fmt.Sprintf("/teams/%s/channels/%s/messages", p.TeamID, p.ChannelID), body)
	}
}

func makeCreateChannel(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *CreateChannelParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *CreateChannelParams) (*mcp.CallToolResult, any, error) {
		if p.TeamID == "" || p.DisplayName == "" {
			return errResult("team_id and display_name are required")
		}
		body := map[string]any{"displayName": p.DisplayName}
		if p.Description != "" {
			body["description"] = p.Description
		}
		return graphPost(auth, fmt.Sprintf("/teams/%s/channels", p.TeamID), body)
	}
}

// --- OneNote Handlers ---

func makeListNotebookPages(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *SectionIDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *SectionIDParam) (*mcp.CallToolResult, any, error) {
		if p.SectionID == "" {
			return errResult("section_id is required")
		}
		return graphGet(auth, fmt.Sprintf("%s/onenote/sections/%s/pages", userPrefix(p.User), p.SectionID))
	}
}

// --- To Do Handlers ---

func makeListTasks(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *TodoListIDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *TodoListIDParam) (*mcp.CallToolResult, any, error) {
		if p.ListID == "" {
			return errResult("list_id is required")
		}
		return graphGet(auth, fmt.Sprintf("%s/todo/lists/%s/tasks", userPrefix(p.User), p.ListID))
	}
}

func makeCreateTask(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *CreateTaskParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *CreateTaskParams) (*mcp.CallToolResult, any, error) {
		if p.ListID == "" || p.Title == "" {
			return errResult("list_id and title are required")
		}
		task := map[string]any{"title": p.Title}
		if p.DueDate != "" {
			task["dueDateTime"] = map[string]any{"dateTime": p.DueDate + "T00:00:00", "timeZone": "UTC"}
		}
		if p.Body != "" {
			task["body"] = map[string]any{"contentType": "text", "content": p.Body}
		}
		return graphPost(auth, fmt.Sprintf("%s/todo/lists/%s/tasks", userPrefix(p.User), p.ListID), task)
	}
}

func makeUpdateTask(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *CreateTaskParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *CreateTaskParams) (*mcp.CallToolResult, any, error) {
		return errResult("use task_id-based update")
	}
}

func makeCompleteTask(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *TaskIDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *TaskIDParam) (*mcp.CallToolResult, any, error) {
		if p.ListID == "" || p.TaskID == "" {
			return errResult("list_id and task_id are required")
		}
		body := map[string]any{"status": "completed"}
		return graphPatch(auth, fmt.Sprintf("%s/todo/lists/%s/tasks/%s", userPrefix(p.User), p.ListID, p.TaskID), body)
	}
}

func makeDeleteTask(auth *tokenProvider) func(context.Context, *mcp.CallToolRequest, *TaskIDParam) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *TaskIDParam) (*mcp.CallToolResult, any, error) {
		if p.ListID == "" || p.TaskID == "" {
			return errResult("list_id and task_id are required")
		}
		return graphDelete(auth, fmt.Sprintf("%s/todo/lists/%s/tasks/%s", userPrefix(p.User), p.ListID, p.TaskID))
	}
}

// --- Graph API Client ---

func graphRequest(auth *tokenProvider, method, path string, body any) (*mcp.CallToolResult, any, error) {
	token, err := auth.getToken()
	if err != nil {
		return errResult("OAuth token error: " + err.Error())
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return errResult(err.Error())
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, graphURL+path, bodyReader)
	if err != nil {
		return errResult(err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return errResult(err.Error())
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errResult(err.Error())
	}

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusAccepted {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "OK"}},
		}, nil, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errResult(fmt.Sprintf("Graph API error %d: %s", resp.StatusCode, string(respBody)))
	}

	var pretty json.RawMessage
	if json.Unmarshal(respBody, &pretty) == nil {
		if indented, err := json.MarshalIndent(pretty, "", "  "); err == nil {
			respBody = indented
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(respBody)}},
	}, nil, nil
}

func graphGet(auth *tokenProvider, path string) (*mcp.CallToolResult, any, error) {
	return graphRequest(auth, "GET", path, nil)
}

func graphPost(auth *tokenProvider, path string, body any) (*mcp.CallToolResult, any, error) {
	return graphRequest(auth, "POST", path, body)
}

func graphPatch(auth *tokenProvider, path string, body any) (*mcp.CallToolResult, any, error) {
	return graphRequest(auth, "PATCH", path, body)
}

func graphDelete(auth *tokenProvider, path string) (*mcp.CallToolResult, any, error) {
	return graphRequest(auth, "DELETE", path, nil)
}

func errResult(msg string) (*mcp.CallToolResult, any, error) {
	r := &mcp.CallToolResult{}
	r.SetError(fmt.Errorf("%s", msg))
	return r, nil, nil
}
