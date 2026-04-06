package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/traffino/mcp-server/internal/config"
	"github.com/traffino/mcp-server/internal/server"
)

const baseURL = "https://api.search.brave.com"

var httpClient = &http.Client{Timeout: 30 * time.Second}

func main() {
	apiKey := config.Require("BRAVE_API_KEY")
	srv := server.New("brave-search", "1.0.0")

	mcp.AddTool(srv.MCPServer(), &mcp.Tool{
		Name:        "web_search",
		Description: "Search the web using Brave Search. Returns web results with titles, URLs, and descriptions.",
	}, makeWebSearchHandler(apiKey))

	mcp.AddTool(srv.MCPServer(), &mcp.Tool{
		Name:        "suggest",
		Description: "Get search suggestions for a partial query using Brave Suggest.",
	}, makeSuggestHandler(apiKey))

	srv.ListenAndServe(config.Get("PORT", ":8000"))
}

func errResult(msg string) (*mcp.CallToolResult, any, error) {
	r := &mcp.CallToolResult{}
	r.SetError(fmt.Errorf("%s", msg))
	return r, nil, nil
}

// --- Web Search ---

type WebSearchParams struct {
	Query      string `json:"query" jsonschema:"Search query (max 400 chars, 50 words)"`
	Count      int    `json:"count,omitempty" jsonschema:"Number of results, 1-20 (default 10)"`
	Offset     int    `json:"offset,omitempty" jsonschema:"Pagination offset, 0-9 (default 0)"`
	Country    string `json:"country,omitempty" jsonschema:"Country code, e.g. US, DE (default US)"`
	Language   string `json:"search_lang,omitempty" jsonschema:"Search language, ISO 639-1 (default en)"`
	Freshness  string `json:"freshness,omitempty" jsonschema:"Time filter: pd (past day), pw (past week), pm (past month), py (past year)"`
	SafeSearch string `json:"safesearch,omitempty" jsonschema:"Safe search: off, moderate, strict (default moderate)"`
}

func makeWebSearchHandler(apiKey string) func(context.Context, *mcp.CallToolRequest, *WebSearchParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, params *WebSearchParams) (*mcp.CallToolResult, any, error) {
		if params.Query == "" {
			return errResult("query is required")
		}

		qp := map[string]string{"q": params.Query}
		if params.Count > 0 {
			qp["count"] = strconv.Itoa(params.Count)
		}
		if params.Offset > 0 {
			qp["offset"] = strconv.Itoa(params.Offset)
		}
		if params.Country != "" {
			qp["country"] = params.Country
		}
		if params.Language != "" {
			qp["search_lang"] = params.Language
		}
		if params.Freshness != "" {
			qp["freshness"] = params.Freshness
		}
		if params.SafeSearch != "" {
			qp["safesearch"] = params.SafeSearch
		}

		body, err := braveGet(apiKey, "/res/v1/web/search", qp)
		if err != nil {
			return errResult(err.Error())
		}

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(body, &raw); err != nil {
			return errResult("failed to parse response")
		}

		results := formatWebResults(raw)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: results}},
		}, nil, nil
	}
}

// --- Suggest ---

type SuggestParams struct {
	Query   string `json:"query" jsonschema:"Partial query for suggestions"`
	Country string `json:"country,omitempty" jsonschema:"Country code, e.g. US, DE"`
	Count   int    `json:"count,omitempty" jsonschema:"Number of suggestions"`
	Rich    bool   `json:"rich,omitempty" jsonschema:"Include entity metadata (default false)"`
}

func makeSuggestHandler(apiKey string) func(context.Context, *mcp.CallToolRequest, *SuggestParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, params *SuggestParams) (*mcp.CallToolResult, any, error) {
		if params.Query == "" {
			return errResult("query is required")
		}

		qp := map[string]string{"q": params.Query}
		if params.Country != "" {
			qp["country"] = params.Country
		}
		if params.Count > 0 {
			qp["count"] = strconv.Itoa(params.Count)
		}
		if params.Rich {
			qp["rich"] = "true"
		}

		body, err := braveGet(apiKey, "/res/v1/suggest/search", qp)
		if err != nil {
			return errResult(err.Error())
		}

		var resp suggestResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return errResult("failed to parse response")
		}

		var lines []string
		for _, s := range resp.Results {
			line := s.Query
			if s.Title != "" {
				line = fmt.Sprintf("%s — %s", s.Query, s.Title)
			}
			lines = append(lines, line)
		}

		text := "No suggestions found."
		if len(lines) > 0 {
			text = strings.Join(lines, "\n")
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, nil, nil
	}
}

// --- API Client ---

func braveGet(apiKey, path string, params map[string]string) ([]byte, error) {
	req, err := http.NewRequest("GET", baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("X-Subscription-Token", apiKey)

	q := req.URL.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("gzip error: %w", err)
		}
		defer gr.Close()
		reader = gr
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Brave API error %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// --- Response Types ---

type suggestResponse struct {
	Results []suggestResult `json:"results"`
}

type suggestResult struct {
	Query       string `json:"query"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

// --- Formatting ---

func formatWebResults(raw map[string]json.RawMessage) string {
	webData, ok := raw["web"]
	if !ok {
		return "No web results found."
	}

	var web struct {
		Results []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
			Age         string `json:"age"`
		} `json:"results"`
	}
	if err := json.Unmarshal(webData, &web); err != nil || len(web.Results) == 0 {
		return "No web results found."
	}

	var sb strings.Builder
	for i, r := range web.Results {
		fmt.Fprintf(&sb, "%d. %s\n   %s\n", i+1, r.Title, r.URL)
		if r.Description != "" {
			fmt.Fprintf(&sb, "   %s\n", r.Description)
		}
		if r.Age != "" {
			fmt.Fprintf(&sb, "   Age: %s\n", r.Age)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
