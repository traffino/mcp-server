package main

import (
	"bytes"
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

const baseURL = "https://discord.com/api/v10"

var httpClient = &http.Client{Timeout: 30 * time.Second}

func main() {
	token := config.Require("DISCORD_BOT_TOKEN")
	srv := server.New("discord", "1.0.0")
	s := srv.MCPServer()

	// Guilds (read)
	mcp.AddTool(s, &mcp.Tool{Name: "list_guilds", Description: "List all guilds the bot is a member of"}, makeListGuilds(token))
	mcp.AddTool(s, &mcp.Tool{Name: "get_guild", Description: "Get guild details by ID"}, makeGetGuild(token))
	mcp.AddTool(s, &mcp.Tool{Name: "list_guild_channels", Description: "List all channels in a guild"}, makeGuildChannels(token))
	mcp.AddTool(s, &mcp.Tool{Name: "list_guild_members", Description: "List members of a guild"}, makeGuildMembers(token))

	// Channels (read + write)
	mcp.AddTool(s, &mcp.Tool{Name: "get_channel", Description: "Get channel details by ID"}, makeGetChannel(token))
	mcp.AddTool(s, &mcp.Tool{Name: "create_channel", Description: "Create a new channel in a guild"}, makeCreateChannel(token))
	mcp.AddTool(s, &mcp.Tool{Name: "edit_channel", Description: "Edit a channel's name or topic"}, makeEditChannel(token))

	// Roles (read)
	mcp.AddTool(s, &mcp.Tool{Name: "list_roles", Description: "List all roles in a guild"}, makeListRoles(token))

	// Reactions (read + write)
	mcp.AddTool(s, &mcp.Tool{Name: "list_reactions", Description: "List users who reacted with a specific emoji"}, makeListReactions(token))
	mcp.AddTool(s, &mcp.Tool{Name: "add_reaction", Description: "Add a reaction to a message"}, makeAddReaction(token))
	mcp.AddTool(s, &mcp.Tool{Name: "remove_reaction", Description: "Remove own reaction from a message"}, makeRemoveReaction(token))

	// Threads (read + write)
	mcp.AddTool(s, &mcp.Tool{Name: "list_active_threads", Description: "List active threads in a guild"}, makeListActiveThreads(token))
	mcp.AddTool(s, &mcp.Tool{Name: "list_archived_threads", Description: "List archived threads in a channel"}, makeListArchivedThreads(token))
	mcp.AddTool(s, &mcp.Tool{Name: "create_thread", Description: "Create a new thread in a channel"}, makeCreateThread(token))
	mcp.AddTool(s, &mcp.Tool{Name: "join_thread", Description: "Join a thread"}, makeJoinThread(token))

	// Users (read)
	mcp.AddTool(s, &mcp.Tool{Name: "get_user", Description: "Get user info by ID"}, makeGetUser(token))
	mcp.AddTool(s, &mcp.Tool{Name: "get_current_user", Description: "Get the bot's own user info"}, makeGetCurrentUser(token))

	srv.ListenAndServe(config.Get("PORT", ":8000"))
}

// --- Param Types ---

type GuildID struct {
	GuildID string `json:"guild_id" jsonschema:"Discord guild (server) ID"`
}

type ChannelID struct {
	ChannelID string `json:"channel_id" jsonschema:"Discord channel ID"`
}

type GuildMembersParams struct {
	GuildID string `json:"guild_id" jsonschema:"Guild ID"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Max members to return, 1-1000 (default 100)"`
}

type CreateChannelParams struct {
	GuildID string `json:"guild_id" jsonschema:"Guild ID"`
	Name    string `json:"name" jsonschema:"Channel name"`
	Type    int    `json:"type,omitempty" jsonschema:"Channel type: 0=text, 2=voice, 4=category, 5=announcement (default 0)"`
	Topic   string `json:"topic,omitempty" jsonschema:"Channel topic"`
}

type EditChannelParams struct {
	ChannelID string `json:"channel_id" jsonschema:"Channel ID"`
	Name      string `json:"name,omitempty" jsonschema:"New channel name"`
	Topic     string `json:"topic,omitempty" jsonschema:"New channel topic"`
}

type ReactionParams struct {
	ChannelID string `json:"channel_id" jsonschema:"Channel ID"`
	MessageID string `json:"message_id" jsonschema:"Message ID"`
	Emoji     string `json:"emoji" jsonschema:"Emoji (Unicode or name:id for custom)"`
}

type ArchivedThreadsParams struct {
	ChannelID string `json:"channel_id" jsonschema:"Channel ID"`
	Type      string `json:"type,omitempty" jsonschema:"Thread type: public or private (default public)"`
}

type CreateThreadParams struct {
	ChannelID string `json:"channel_id" jsonschema:"Channel ID to create thread in"`
	Name      string `json:"name" jsonschema:"Thread name"`
	Type      int    `json:"type,omitempty" jsonschema:"Thread type: 11=public, 12=private (default 11)"`
}

type UserID struct {
	UserID string `json:"user_id" jsonschema:"Discord user ID"`
}

// --- Handlers ---

func makeListGuilds(token string) func(context.Context, *mcp.CallToolRequest, any) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
		return discordGet(token, "/users/@me/guilds")
	}
}

func makeGetGuild(token string) func(context.Context, *mcp.CallToolRequest, *GuildID) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *GuildID) (*mcp.CallToolResult, any, error) {
		if p.GuildID == "" {
			return errResult("guild_id is required")
		}
		return discordGet(token, fmt.Sprintf("/guilds/%s", p.GuildID))
	}
}

func makeGuildChannels(token string) func(context.Context, *mcp.CallToolRequest, *GuildID) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *GuildID) (*mcp.CallToolResult, any, error) {
		if p.GuildID == "" {
			return errResult("guild_id is required")
		}
		return discordGet(token, fmt.Sprintf("/guilds/%s/channels", p.GuildID))
	}
}

func makeGuildMembers(token string) func(context.Context, *mcp.CallToolRequest, *GuildMembersParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *GuildMembersParams) (*mcp.CallToolResult, any, error) {
		if p.GuildID == "" {
			return errResult("guild_id is required")
		}
		path := fmt.Sprintf("/guilds/%s/members?limit=%d", p.GuildID, max(p.Limit, 100))
		return discordGet(token, path)
	}
}

func makeGetChannel(token string) func(context.Context, *mcp.CallToolRequest, *ChannelID) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ChannelID) (*mcp.CallToolResult, any, error) {
		if p.ChannelID == "" {
			return errResult("channel_id is required")
		}
		return discordGet(token, fmt.Sprintf("/channels/%s", p.ChannelID))
	}
}

func makeCreateChannel(token string) func(context.Context, *mcp.CallToolRequest, *CreateChannelParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *CreateChannelParams) (*mcp.CallToolResult, any, error) {
		if p.GuildID == "" || p.Name == "" {
			return errResult("guild_id and name are required")
		}
		body := map[string]any{"name": p.Name}
		if p.Type > 0 {
			body["type"] = p.Type
		}
		if p.Topic != "" {
			body["topic"] = p.Topic
		}
		return discordPost(token, fmt.Sprintf("/guilds/%s/channels", p.GuildID), body)
	}
}

func makeEditChannel(token string) func(context.Context, *mcp.CallToolRequest, *EditChannelParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *EditChannelParams) (*mcp.CallToolResult, any, error) {
		if p.ChannelID == "" {
			return errResult("channel_id is required")
		}
		body := map[string]any{}
		if p.Name != "" {
			body["name"] = p.Name
		}
		if p.Topic != "" {
			body["topic"] = p.Topic
		}
		return discordPatch(token, fmt.Sprintf("/channels/%s", p.ChannelID), body)
	}
}

func makeListRoles(token string) func(context.Context, *mcp.CallToolRequest, *GuildID) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *GuildID) (*mcp.CallToolResult, any, error) {
		if p.GuildID == "" {
			return errResult("guild_id is required")
		}
		return discordGet(token, fmt.Sprintf("/guilds/%s/roles", p.GuildID))
	}
}

func makeListReactions(token string) func(context.Context, *mcp.CallToolRequest, *ReactionParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ReactionParams) (*mcp.CallToolResult, any, error) {
		if p.ChannelID == "" || p.MessageID == "" || p.Emoji == "" {
			return errResult("channel_id, message_id, and emoji are required")
		}
		return discordGet(token, fmt.Sprintf("/channels/%s/messages/%s/reactions/%s", p.ChannelID, p.MessageID, p.Emoji))
	}
}

func makeAddReaction(token string) func(context.Context, *mcp.CallToolRequest, *ReactionParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ReactionParams) (*mcp.CallToolResult, any, error) {
		if p.ChannelID == "" || p.MessageID == "" || p.Emoji == "" {
			return errResult("channel_id, message_id, and emoji are required")
		}
		return discordPut(token, fmt.Sprintf("/channels/%s/messages/%s/reactions/%s/@me", p.ChannelID, p.MessageID, p.Emoji))
	}
}

func makeRemoveReaction(token string) func(context.Context, *mcp.CallToolRequest, *ReactionParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ReactionParams) (*mcp.CallToolResult, any, error) {
		if p.ChannelID == "" || p.MessageID == "" || p.Emoji == "" {
			return errResult("channel_id, message_id, and emoji are required")
		}
		return discordDelete(token, fmt.Sprintf("/channels/%s/messages/%s/reactions/%s/@me", p.ChannelID, p.MessageID, p.Emoji))
	}
}

func makeListActiveThreads(token string) func(context.Context, *mcp.CallToolRequest, *GuildID) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *GuildID) (*mcp.CallToolResult, any, error) {
		if p.GuildID == "" {
			return errResult("guild_id is required")
		}
		return discordGet(token, fmt.Sprintf("/guilds/%s/threads/active", p.GuildID))
	}
}

func makeListArchivedThreads(token string) func(context.Context, *mcp.CallToolRequest, *ArchivedThreadsParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ArchivedThreadsParams) (*mcp.CallToolResult, any, error) {
		if p.ChannelID == "" {
			return errResult("channel_id is required")
		}
		threadType := "public"
		if p.Type == "private" {
			threadType = "private"
		}
		return discordGet(token, fmt.Sprintf("/channels/%s/threads/archived/%s", p.ChannelID, threadType))
	}
}

func makeCreateThread(token string) func(context.Context, *mcp.CallToolRequest, *CreateThreadParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *CreateThreadParams) (*mcp.CallToolResult, any, error) {
		if p.ChannelID == "" || p.Name == "" {
			return errResult("channel_id and name are required")
		}
		threadType := 11
		if p.Type == 12 {
			threadType = 12
		}
		body := map[string]any{"name": p.Name, "type": threadType}
		return discordPost(token, fmt.Sprintf("/channels/%s/threads", p.ChannelID), body)
	}
}

func makeJoinThread(token string) func(context.Context, *mcp.CallToolRequest, *ChannelID) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ChannelID) (*mcp.CallToolResult, any, error) {
		if p.ChannelID == "" {
			return errResult("channel_id (thread ID) is required")
		}
		return discordPut(token, fmt.Sprintf("/channels/%s/thread-members/@me", p.ChannelID))
	}
}

func makeGetUser(token string) func(context.Context, *mcp.CallToolRequest, *UserID) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *UserID) (*mcp.CallToolResult, any, error) {
		if p.UserID == "" {
			return errResult("user_id is required")
		}
		return discordGet(token, fmt.Sprintf("/users/%s", p.UserID))
	}
}

func makeGetCurrentUser(token string) func(context.Context, *mcp.CallToolRequest, any) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
		return discordGet(token, "/users/@me")
	}
}

// --- API Client ---

func discordRequest(token, method, path string, body any) (*mcp.CallToolResult, any, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return errResult(err.Error())
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		return errResult(err.Error())
	}

	req.Header.Set("Authorization", "Bot "+token)
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

	// 204 No Content is a success for PUT/DELETE reactions
	if resp.StatusCode == http.StatusNoContent {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "OK"}},
		}, nil, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errResult(fmt.Sprintf("Discord API error %d: %s", resp.StatusCode, string(respBody)))
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

func discordGet(token, path string) (*mcp.CallToolResult, any, error) {
	return discordRequest(token, "GET", path, nil)
}

func discordPost(token, path string, body any) (*mcp.CallToolResult, any, error) {
	return discordRequest(token, "POST", path, body)
}

func discordPatch(token, path string, body any) (*mcp.CallToolResult, any, error) {
	return discordRequest(token, "PATCH", path, body)
}

func discordPut(token, path string) (*mcp.CallToolResult, any, error) {
	return discordRequest(token, "PUT", path, nil)
}

func discordDelete(token, path string) (*mcp.CallToolResult, any, error) {
	return discordRequest(token, "DELETE", path, nil)
}

func errResult(msg string) (*mcp.CallToolResult, any, error) {
	r := &mcp.CallToolResult{}
	r.SetError(fmt.Errorf("%s", msg))
	return r, nil, nil
}
