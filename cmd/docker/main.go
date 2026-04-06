package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/traffino/mcp-server/internal/config"
	"github.com/traffino/mcp-server/internal/server"
)

var dockerClient *http.Client

func main() {
	socketPath := config.Get("DOCKER_HOST", "/var/run/docker.sock")
	dockerClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	srv := server.New("docker", "1.0.0")
	s := srv.MCPServer()

	// Containers - read
	mcp.AddTool(s, &mcp.Tool{Name: "list_containers", Description: "List all containers (including stopped)"}, makeListContainers())
	mcp.AddTool(s, &mcp.Tool{Name: "inspect_container", Description: "Get detailed container info"}, makeInspectContainer())
	mcp.AddTool(s, &mcp.Tool{Name: "container_logs", Description: "Get container logs"}, makeContainerLogs())
	mcp.AddTool(s, &mcp.Tool{Name: "container_stats", Description: "Get container resource usage stats"}, makeContainerStats())
	mcp.AddTool(s, &mcp.Tool{Name: "container_top", Description: "List processes in a container"}, makeContainerTop())

	// Containers - write
	mcp.AddTool(s, &mcp.Tool{Name: "create_container", Description: "Create a new container"}, makeCreateContainer())
	mcp.AddTool(s, &mcp.Tool{Name: "start_container", Description: "Start a stopped container"}, makeContainerAction("start"))
	mcp.AddTool(s, &mcp.Tool{Name: "stop_container", Description: "Stop a running container"}, makeContainerAction("stop"))
	mcp.AddTool(s, &mcp.Tool{Name: "restart_container", Description: "Restart a container"}, makeContainerAction("restart"))
	mcp.AddTool(s, &mcp.Tool{Name: "remove_container", Description: "Remove a container"}, makeRemoveContainer())

	// Images - read
	mcp.AddTool(s, &mcp.Tool{Name: "list_images", Description: "List all images"}, makeListImages())
	mcp.AddTool(s, &mcp.Tool{Name: "inspect_image", Description: "Get detailed image info"}, makeInspectImage())

	// Images - write
	mcp.AddTool(s, &mcp.Tool{Name: "pull_image", Description: "Pull an image from a registry"}, makePullImage())
	mcp.AddTool(s, &mcp.Tool{Name: "remove_image", Description: "Remove an image"}, makeRemoveImage())
	mcp.AddTool(s, &mcp.Tool{Name: "prune_images", Description: "Remove unused images"}, makePruneImages())

	// Networks - read
	mcp.AddTool(s, &mcp.Tool{Name: "list_networks", Description: "List all networks"}, makeDockerGet("/networks"))
	mcp.AddTool(s, &mcp.Tool{Name: "inspect_network", Description: "Get detailed network info"}, makeDockerGetByID("/networks"))

	// Networks - write
	mcp.AddTool(s, &mcp.Tool{Name: "create_network", Description: "Create a network"}, makeCreateNetwork())
	mcp.AddTool(s, &mcp.Tool{Name: "remove_network", Description: "Remove a network"}, makeDockerDeleteByID("/networks"))

	// Volumes - read
	mcp.AddTool(s, &mcp.Tool{Name: "list_volumes", Description: "List all volumes"}, makeDockerGet("/volumes"))
	mcp.AddTool(s, &mcp.Tool{Name: "inspect_volume", Description: "Get detailed volume info"}, makeDockerGetByID("/volumes"))

	// Volumes - write
	mcp.AddTool(s, &mcp.Tool{Name: "create_volume", Description: "Create a volume"}, makeCreateVolume())
	mcp.AddTool(s, &mcp.Tool{Name: "remove_volume", Description: "Remove a volume"}, makeDockerDeleteByID("/volumes"))
	mcp.AddTool(s, &mcp.Tool{Name: "prune_volumes", Description: "Remove unused volumes"}, makePruneVolumes())

	// System
	mcp.AddTool(s, &mcp.Tool{Name: "system_info", Description: "Get Docker system info"}, makeDockerGetNoParams("/info"))
	mcp.AddTool(s, &mcp.Tool{Name: "system_version", Description: "Get Docker version"}, makeDockerGetNoParams("/version"))
	mcp.AddTool(s, &mcp.Tool{Name: "system_df", Description: "Get Docker disk usage"}, makeDockerGetNoParams("/system/df"))
	mcp.AddTool(s, &mcp.Tool{Name: "system_ping", Description: "Ping Docker daemon"}, makeDockerGetNoParams("/_ping"))

	srv.ListenAndServe(config.Get("PORT", ":8000"))
}

// --- Param Types ---

type ContainerID struct {
	ID string `json:"id" jsonschema:"Container ID or name"`
}

type LogsParams struct {
	ID     string `json:"id" jsonschema:"Container ID or name"`
	Tail   string `json:"tail,omitempty" jsonschema:"Number of lines from end (default all)"`
	Since  string `json:"since,omitempty" jsonschema:"Show logs since timestamp or relative (e.g. 10m)"`
	Stderr bool   `json:"stderr,omitempty" jsonschema:"Include stderr (default true)"`
	Stdout bool   `json:"stdout,omitempty" jsonschema:"Include stdout (default true)"`
}

type CreateContainerParams struct {
	Image   string            `json:"image" jsonschema:"Image name"`
	Name    string            `json:"name,omitempty" jsonschema:"Container name"`
	Env     []string          `json:"env,omitempty" jsonschema:"Environment variables (KEY=VALUE)"`
	Cmd     []string          `json:"cmd,omitempty" jsonschema:"Command to run"`
	Labels  map[string]string `json:"labels,omitempty" jsonschema:"Container labels"`
	Restart string            `json:"restart,omitempty" jsonschema:"Restart policy: no, always, unless-stopped, on-failure"`
}

type ImageRef struct {
	Image string `json:"image" jsonschema:"Image name with optional tag (e.g. nginx:latest)"`
}

type NameOnly struct {
	Name string `json:"name" jsonschema:"Resource name"`
}

type IDOnly struct {
	ID string `json:"id" jsonschema:"Resource ID or name"`
}

type CreateNetworkParams struct {
	Name   string `json:"name" jsonschema:"Network name"`
	Driver string `json:"driver,omitempty" jsonschema:"Network driver (default bridge)"`
}

type CreateVolumeParams struct {
	Name   string `json:"name" jsonschema:"Volume name"`
	Driver string `json:"driver,omitempty" jsonschema:"Volume driver (default local)"`
}

// --- Handlers ---

func makeListContainers() func(context.Context, *mcp.CallToolRequest, any) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
		return dockerGet("/containers/json?all=true")
	}
}

func makeInspectContainer() func(context.Context, *mcp.CallToolRequest, *ContainerID) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ContainerID) (*mcp.CallToolResult, any, error) {
		if p.ID == "" {
			return errResult("id is required")
		}
		return dockerGet(fmt.Sprintf("/containers/%s/json", p.ID))
	}
}

func makeContainerLogs() func(context.Context, *mcp.CallToolRequest, *LogsParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *LogsParams) (*mcp.CallToolResult, any, error) {
		if p.ID == "" {
			return errResult("id is required")
		}
		path := fmt.Sprintf("/containers/%s/logs?stdout=true&stderr=true", p.ID)
		if p.Tail != "" {
			path += "&tail=" + p.Tail
		}
		if p.Since != "" {
			path += "&since=" + p.Since
		}
		return dockerGet(path)
	}
}

func makeContainerStats() func(context.Context, *mcp.CallToolRequest, *ContainerID) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ContainerID) (*mcp.CallToolResult, any, error) {
		if p.ID == "" {
			return errResult("id is required")
		}
		return dockerGet(fmt.Sprintf("/containers/%s/stats?stream=false", p.ID))
	}
}

func makeContainerTop() func(context.Context, *mcp.CallToolRequest, *ContainerID) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ContainerID) (*mcp.CallToolResult, any, error) {
		if p.ID == "" {
			return errResult("id is required")
		}
		return dockerGet(fmt.Sprintf("/containers/%s/top", p.ID))
	}
}

func makeCreateContainer() func(context.Context, *mcp.CallToolRequest, *CreateContainerParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *CreateContainerParams) (*mcp.CallToolResult, any, error) {
		if p.Image == "" {
			return errResult("image is required")
		}
		body := map[string]any{"Image": p.Image}
		if len(p.Env) > 0 {
			body["Env"] = p.Env
		}
		if len(p.Cmd) > 0 {
			body["Cmd"] = p.Cmd
		}
		if len(p.Labels) > 0 {
			body["Labels"] = p.Labels
		}
		if p.Restart != "" {
			body["HostConfig"] = map[string]any{"RestartPolicy": map[string]any{"Name": p.Restart}}
		}
		path := "/containers/create"
		if p.Name != "" {
			path += "?name=" + p.Name
		}
		return dockerPost(path, body)
	}
}

func makeContainerAction(action string) func(context.Context, *mcp.CallToolRequest, *ContainerID) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ContainerID) (*mcp.CallToolResult, any, error) {
		if p.ID == "" {
			return errResult("id is required")
		}
		return dockerPost(fmt.Sprintf("/containers/%s/%s", p.ID, action), nil)
	}
}

func makeRemoveContainer() func(context.Context, *mcp.CallToolRequest, *ContainerID) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ContainerID) (*mcp.CallToolResult, any, error) {
		if p.ID == "" {
			return errResult("id is required")
		}
		return dockerDelete(fmt.Sprintf("/containers/%s?force=true", p.ID))
	}
}

func makeListImages() func(context.Context, *mcp.CallToolRequest, any) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
		return dockerGet("/images/json")
	}
}

func makeInspectImage() func(context.Context, *mcp.CallToolRequest, *ImageRef) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ImageRef) (*mcp.CallToolResult, any, error) {
		if p.Image == "" {
			return errResult("image is required")
		}
		return dockerGet(fmt.Sprintf("/images/%s/json", p.Image))
	}
}

func makePullImage() func(context.Context, *mcp.CallToolRequest, *ImageRef) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ImageRef) (*mcp.CallToolResult, any, error) {
		if p.Image == "" {
			return errResult("image is required")
		}
		return dockerPost(fmt.Sprintf("/images/create?fromImage=%s", p.Image), nil)
	}
}

func makeRemoveImage() func(context.Context, *mcp.CallToolRequest, *ImageRef) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ImageRef) (*mcp.CallToolResult, any, error) {
		if p.Image == "" {
			return errResult("image is required")
		}
		return dockerDelete(fmt.Sprintf("/images/%s", p.Image))
	}
}

func makePruneImages() func(context.Context, *mcp.CallToolRequest, any) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
		return dockerPost("/images/prune", nil)
	}
}

func makeDockerGet(path string) func(context.Context, *mcp.CallToolRequest, any) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
		return dockerGet(path)
	}
}

func makeDockerGetNoParams(path string) func(context.Context, *mcp.CallToolRequest, any) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
		return dockerGet(path)
	}
}

func makeDockerGetByID(basePath string) func(context.Context, *mcp.CallToolRequest, *IDOnly) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IDOnly) (*mcp.CallToolResult, any, error) {
		if p.ID == "" {
			return errResult("id is required")
		}
		return dockerGet(fmt.Sprintf("%s/%s", basePath, p.ID))
	}
}

func makeDockerDeleteByID(basePath string) func(context.Context, *mcp.CallToolRequest, *IDOnly) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IDOnly) (*mcp.CallToolResult, any, error) {
		if p.ID == "" {
			return errResult("id is required")
		}
		return dockerDelete(fmt.Sprintf("%s/%s", basePath, p.ID))
	}
}

func makeCreateNetwork() func(context.Context, *mcp.CallToolRequest, *CreateNetworkParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *CreateNetworkParams) (*mcp.CallToolResult, any, error) {
		if p.Name == "" {
			return errResult("name is required")
		}
		body := map[string]any{"Name": p.Name}
		if p.Driver != "" {
			body["Driver"] = p.Driver
		}
		return dockerPost("/networks/create", body)
	}
}

func makeCreateVolume() func(context.Context, *mcp.CallToolRequest, *CreateVolumeParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *CreateVolumeParams) (*mcp.CallToolResult, any, error) {
		if p.Name == "" {
			return errResult("name is required")
		}
		body := map[string]any{"Name": p.Name}
		if p.Driver != "" {
			body["Driver"] = p.Driver
		}
		return dockerPost("/volumes/create", body)
	}
}

func makePruneVolumes() func(context.Context, *mcp.CallToolRequest, any) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
		return dockerPost("/volumes/prune", nil)
	}
}

// --- Docker API Client ---

func dockerGet(path string) (*mcp.CallToolResult, any, error) {
	return dockerRequest("GET", path, nil)
}

func dockerPost(path string, body any) (*mcp.CallToolResult, any, error) {
	return dockerRequest("POST", path, body)
}

func dockerDelete(path string) (*mcp.CallToolResult, any, error) {
	return dockerRequest("DELETE", path, nil)
}

func dockerRequest(method, path string, body any) (*mcp.CallToolResult, any, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return errResult(err.Error())
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, "http://localhost"+path, bodyReader)
	if err != nil {
		return errResult(err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := dockerClient.Do(req)
	if err != nil {
		return errResult(err.Error())
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errResult(err.Error())
	}

	if resp.StatusCode == http.StatusNoContent {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "OK"}},
		}, nil, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errResult(fmt.Sprintf("Docker API error %d: %s", resp.StatusCode, string(respBody)))
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

func errResult(msg string) (*mcp.CallToolResult, any, error) {
	r := &mcp.CallToolResult{}
	r.SetError(fmt.Errorf("%s", msg))
	return r, nil, nil
}
