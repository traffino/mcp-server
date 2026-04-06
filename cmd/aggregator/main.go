package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/traffino/mcp-server/internal/config"
)

const protocolVersion = "2024-11-05"
const requestTimeout = 15 * time.Second

var httpClient = &http.Client{Timeout: requestTimeout}

func main() {
	backends := config.Require("MCP_BACKENDS")
	addr := config.Get("PORT", ":8080")
	if len(addr) > 0 && addr[0] != ':' {
		addr = ":" + addr
	}

	agg := newAggregator()
	agg.initAll(backends)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","backends":%d,"tools":%d}`, agg.backendCount(), agg.toolCount())
	})

	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		raw, err := io.ReadAll(r.Body)
		if err != nil {
			writeJSONRPCError(w, nil, -32700, "Parse error")
			return
		}
		defer r.Body.Close()

		var req jsonRPCRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			writeJSONRPCError(w, nil, -32700, "Parse error")
			return
		}

		result := agg.handleRequest(&req)
		if result == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		headers := w.Header()
		headers.Set("Content-Type", "application/json")
		if agg.sessionID != "" {
			headers.Set("Mcp-Session-Id", agg.sessionID)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	})

	log.Printf("[aggregator] listening on %s with %d backends, %d tools", addr, agg.backendCount(), agg.toolCount())
	log.Fatal(http.ListenAndServe(addr, mux))
}

// --- JSON-RPC Types ---

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// --- Backend ---

type backend struct {
	name      string
	url       string
	sessionID string
}

func (b *backend) request(method string, params any, id any) (*jsonRPCResponse, error) {
	body := map[string]any{"jsonrpc": "2.0", "method": method}
	if params != nil {
		body["params"] = params
	}
	if id != nil {
		body["id"] = id
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", b.url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if b.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", b.sessionID)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	sid := resp.Header.Get("Mcp-Session-Id")
	if sid != "" {
		b.sessionID = sid
	}

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusAccepted {
		return nil, nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "text/event-stream") {
		return parseSSE(respBody)
	}

	var result jsonRPCResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("invalid response: %s", string(respBody))
	}
	return &result, nil
}

func parseSSE(data []byte) (*jsonRPCResponse, error) {
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "data: ") {
			var resp jsonRPCResponse
			if err := json.Unmarshal([]byte(line[6:]), &resp); err == nil {
				return &resp, nil
			}
		}
	}
	return nil, fmt.Errorf("no JSON-RPC in SSE response")
}

// --- Aggregator ---

type toolEntry struct {
	backend      *backend
	originalName string
}

type aggregator struct {
	backends  map[string]*backend
	tools     []map[string]any
	toolMap   map[string]*toolEntry
	sessionID string
	mu        sync.RWMutex
}

func newAggregator() *aggregator {
	return &aggregator{
		backends: make(map[string]*backend),
		toolMap:  make(map[string]*toolEntry),
	}
}

func (a *aggregator) backendCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.backends)
}

func (a *aggregator) toolCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.tools)
}

func (a *aggregator) initAll(spec string) {
	entries := parseBackends(spec)
	log.Printf("Initializing %d backend(s)...", len(entries))

	var wg sync.WaitGroup
	for _, e := range entries {
		wg.Add(1)
		go func(name, url string) {
			defer wg.Done()
			if err := a.initBackend(name, url); err != nil {
				log.Printf("[%s] init failed: %v", name, err)
			}
		}(e.name, e.url)
	}
	wg.Wait()

	log.Printf("Ready. %d tool(s) from %d backend(s).", a.toolCount(), a.backendCount())
}

func (a *aggregator) initBackend(name, url string) error {
	b := &backend{name: name, url: url}

	initResult, err := b.request("initialize", map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "mcp-aggregator", "version": "1.0.0"},
	}, 1)
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	serverInfo := ""
	if initResult != nil && initResult.Result != nil {
		if m, ok := initResult.Result.(map[string]any); ok {
			if si, ok := m["serverInfo"].(map[string]any); ok {
				serverInfo = fmt.Sprintf("%v %v", si["name"], si["version"])
			}
		}
	}
	log.Printf("[%s] Server: %s", name, serverInfo)

	b.request("notifications/initialized", map[string]any{}, nil)

	toolsResult, err := b.request("tools/list", map[string]any{}, 2)
	if err != nil {
		return fmt.Errorf("tools/list: %w", err)
	}

	var tools []map[string]any
	if toolsResult != nil && toolsResult.Result != nil {
		if m, ok := toolsResult.Result.(map[string]any); ok {
			if t, ok := m["tools"].([]any); ok {
				for _, tool := range t {
					if tm, ok := tool.(map[string]any); ok {
						tools = append(tools, tm)
					}
				}
			}
		}
	}

	log.Printf("[%s] %d tools discovered", name, len(tools))

	a.mu.Lock()
	defer a.mu.Unlock()
	a.backends[name] = b

	for _, tool := range tools {
		origName, _ := tool["name"].(string)
		prefixed := name + "_" + origName
		desc, _ := tool["description"].(string)

		a.toolMap[prefixed] = &toolEntry{backend: b, originalName: origName}
		a.tools = append(a.tools, map[string]any{
			"name":        prefixed,
			"description": fmt.Sprintf("[%s] %s", name, desc),
			"inputSchema": tool["inputSchema"],
		})
	}

	return nil
}

func (a *aggregator) handleRequest(req *jsonRPCRequest) *jsonRPCResponse {
	switch req.Method {
	case "initialize":
		a.sessionID = fmt.Sprintf("agg-%d", time.Now().UnixMilli())
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": protocolVersion,
				"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
				"serverInfo":      map[string]any{"name": "mcp-aggregator", "version": "1.0.0"},
			},
		}

	case "notifications/initialized":
		return nil

	case "tools/list":
		a.mu.RLock()
		tools := a.tools
		a.mu.RUnlock()
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{"tools": tools},
		}

	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &jsonRPCError{Code: -32602, Message: "Invalid params"}}
		}

		a.mu.RLock()
		entry, ok := a.toolMap[params.Name]
		a.mu.RUnlock()

		if !ok {
			return &jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &jsonRPCError{Code: -32602, Message: "Unknown tool: " + params.Name}}
		}

		result, err := entry.backend.request("tools/call", map[string]any{
			"name":      entry.originalName,
			"arguments": params.Arguments,
		}, time.Now().UnixMilli())

		if err != nil {
			return &jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &jsonRPCError{Code: -32603, Message: "Backend error: " + err.Error()}}
		}

		return &jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: result.Result}

	default:
		return &jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &jsonRPCError{Code: -32601, Message: "Method not found: " + req.Method}}
	}
}

// --- Helpers ---

type backendEntry struct {
	name string
	url  string
}

func parseBackends(spec string) []backendEntry {
	var entries []backendEntry
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, "=")
		if idx < 0 {
			log.Printf("Invalid backend spec: %s", part)
			continue
		}
		entries = append(entries, backendEntry{
			name: strings.TrimSpace(part[:idx]),
			url:  strings.TrimSpace(part[idx+1:]),
		})
	}
	return entries
}

func writeJSONRPCError(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &jsonRPCError{Code: code, Message: msg},
	})
}
