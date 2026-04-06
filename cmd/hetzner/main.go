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

const baseURL = "https://api.hetzner.cloud/v1"

var httpClient = &http.Client{Timeout: 30 * time.Second}

func main() {
	apiToken := config.Require("HETZNER_API_TOKEN")
	srv := server.New("hetzner-cloud", "1.0.0")
	s := srv.MCPServer()

	// Servers
	addListTool(s, apiToken, "list_servers", "List all servers", "/servers", "servers")
	addGetTool(s, apiToken, "get_server", "Get server details by ID", "/servers")
	addListTool(s, apiToken, "list_server_types", "List available server types", "/server_types", "server_types")

	// SSH Keys
	addListTool(s, apiToken, "list_ssh_keys", "List all SSH keys", "/ssh_keys", "ssh_keys")
	addGetTool(s, apiToken, "get_ssh_key", "Get SSH key by ID", "/ssh_keys")

	// Firewalls
	addListTool(s, apiToken, "list_firewalls", "List all firewalls", "/firewalls", "firewalls")
	addGetTool(s, apiToken, "get_firewall", "Get firewall details by ID", "/firewalls")

	// Networks
	addListTool(s, apiToken, "list_networks", "List all networks", "/networks", "networks")
	addGetTool(s, apiToken, "get_network", "Get network details by ID", "/networks")

	// Volumes
	addListTool(s, apiToken, "list_volumes", "List all volumes", "/volumes", "volumes")
	addGetTool(s, apiToken, "get_volume", "Get volume details by ID", "/volumes")

	// Floating IPs
	addListTool(s, apiToken, "list_floating_ips", "List all floating IPs", "/floating_ips", "floating_ips")
	addGetTool(s, apiToken, "get_floating_ip", "Get floating IP by ID", "/floating_ips")

	// Images
	addListTool(s, apiToken, "list_images", "List all images (OS, snapshots, backups)", "/images", "images")
	addGetTool(s, apiToken, "get_image", "Get image details by ID", "/images")

	// Locations & Datacenters
	addListTool(s, apiToken, "list_locations", "List all locations", "/locations", "locations")
	addGetTool(s, apiToken, "get_location", "Get location by ID", "/locations")
	addListTool(s, apiToken, "list_datacenters", "List all datacenters", "/datacenters", "datacenters")
	addGetTool(s, apiToken, "get_datacenter", "Get datacenter by ID", "/datacenters")

	// Load Balancers
	addListTool(s, apiToken, "list_load_balancers", "List all load balancers", "/load_balancers", "load_balancers")
	addGetTool(s, apiToken, "get_load_balancer", "Get load balancer by ID", "/load_balancers")
	addListTool(s, apiToken, "list_load_balancer_types", "List load balancer types", "/load_balancer_types", "load_balancer_types")

	// Certificates
	addListTool(s, apiToken, "list_certificates", "List all certificates", "/certificates", "certificates")
	addGetTool(s, apiToken, "get_certificate", "Get certificate by ID", "/certificates")

	// Primary IPs
	addListTool(s, apiToken, "list_primary_ips", "List all primary IPs", "/primary_ips", "primary_ips")
	addGetTool(s, apiToken, "get_primary_ip", "Get primary IP by ID", "/primary_ips")

	// Placement Groups
	addListTool(s, apiToken, "list_placement_groups", "List all placement groups", "/placement_groups", "placement_groups")
	addGetTool(s, apiToken, "get_placement_group", "Get placement group by ID", "/placement_groups")

	// Actions
	addListTool(s, apiToken, "list_actions", "List all actions", "/actions", "actions")
	addGetTool(s, apiToken, "get_action", "Get action details by ID", "/actions")

	// Pricing
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_pricing",
		Description: "Get current pricing information for all resources",
	}, makePricingHandler(apiToken))

	// Server Metrics
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_server_metrics",
		Description: "Get metrics (CPU, disk, network) for a server",
	}, makeMetricsHandler(apiToken))

	srv.ListenAndServe(config.Get("PORT", ":8000"))
}

// --- Generic List/Get Tools ---

type ListParams struct {
	Page    int    `json:"page,omitempty" jsonschema:"Page number (default 1)"`
	PerPage int    `json:"per_page,omitempty" jsonschema:"Items per page, max 50 (default 25)"`
	Sort    string `json:"sort,omitempty" jsonschema:"Sort field, e.g. id:asc, name:desc"`
	Name    string `json:"name,omitempty" jsonschema:"Filter by name"`
}

type GetParams struct {
	ID int64 `json:"id" jsonschema:"Resource ID"`
}

func addListTool(s *mcp.Server, token, name, desc, path, key string) {
	mcp.AddTool(s, &mcp.Tool{Name: name, Description: desc},
		func(ctx context.Context, req *mcp.CallToolRequest, params *ListParams) (*mcp.CallToolResult, any, error) {
			qp := map[string]string{}
			if params.Page > 0 {
				qp["page"] = fmt.Sprintf("%d", params.Page)
			}
			if params.PerPage > 0 {
				qp["per_page"] = fmt.Sprintf("%d", params.PerPage)
			}
			if params.Sort != "" {
				qp["sort"] = params.Sort
			}
			if params.Name != "" {
				qp["name"] = params.Name
			}
			return hetznerGet(token, path, qp)
		})
}

func addGetTool(s *mcp.Server, token, name, desc, path string) {
	mcp.AddTool(s, &mcp.Tool{Name: name, Description: desc},
		func(ctx context.Context, req *mcp.CallToolRequest, params *GetParams) (*mcp.CallToolResult, any, error) {
			if params.ID <= 0 {
				return errResult("id is required")
			}
			return hetznerGet(token, fmt.Sprintf("%s/%d", path, params.ID), nil)
		})
}

// --- Special Tools ---

type MetricsParams struct {
	ServerID int64  `json:"server_id" jsonschema:"Server ID"`
	Type     string `json:"type" jsonschema:"Metric type: cpu, disk, network"`
	Start    string `json:"start" jsonschema:"Start time ISO 8601, e.g. 2024-01-01T00:00:00Z"`
	End      string `json:"end" jsonschema:"End time ISO 8601, e.g. 2024-01-02T00:00:00Z"`
	Step     int    `json:"step,omitempty" jsonschema:"Step in seconds (default auto)"`
}

func makeMetricsHandler(token string) func(context.Context, *mcp.CallToolRequest, *MetricsParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, params *MetricsParams) (*mcp.CallToolResult, any, error) {
		if params.ServerID <= 0 || params.Type == "" || params.Start == "" || params.End == "" {
			return errResult("server_id, type, start, and end are required")
		}
		qp := map[string]string{
			"type":  params.Type,
			"start": params.Start,
			"end":   params.End,
		}
		if params.Step > 0 {
			qp["step"] = fmt.Sprintf("%d", params.Step)
		}
		return hetznerGet(token, fmt.Sprintf("/servers/%d/metrics", params.ServerID), qp)
	}
}

func makePricingHandler(token string) func(context.Context, *mcp.CallToolRequest, any) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
		return hetznerGet(token, "/pricing", nil)
	}
}

// --- API Client ---

func hetznerGet(token, path string, params map[string]string) (*mcp.CallToolResult, any, error) {
	req, err := http.NewRequest("GET", baseURL+path, nil)
	if err != nil {
		return errResult(err.Error())
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	q := req.URL.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()

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
		return errResult(fmt.Sprintf("Hetzner API error %d: %s", resp.StatusCode, string(body)))
	}

	// Pretty-print JSON for readability
	var pretty json.RawMessage
	if json.Unmarshal(body, &pretty) == nil {
		indented, err := json.MarshalIndent(pretty, "", "  ")
		if err == nil {
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
