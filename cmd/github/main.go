package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/traffino/mcp-server/internal/config"
	"github.com/traffino/mcp-server/internal/server"
)

const baseURL = "https://api.github.com"

var httpClient = &http.Client{Timeout: 30 * time.Second}

func main() {
	token := config.Require("GITHUB_PERSONAL_ACCESS_TOKEN")
	srv := server.New("github", "1.0.0")
	s := srv.MCPServer()

	// Repos
	mcp.AddTool(s, &mcp.Tool{Name: "list_repos", Description: "List repositories for a user or organization"}, makeListRepos(token))
	mcp.AddTool(s, &mcp.Tool{Name: "get_repo", Description: "Get repository details"}, makeGetRepo(token))
	mcp.AddTool(s, &mcp.Tool{Name: "list_branches", Description: "List branches of a repository"}, makeRepoSublist(token, "branches"))
	mcp.AddTool(s, &mcp.Tool{Name: "list_commits", Description: "List commits of a repository"}, makeRepoSublist(token, "commits"))
	mcp.AddTool(s, &mcp.Tool{Name: "get_repo_content", Description: "Get file or directory content from a repository"}, makeGetContent(token))

	// Issues
	mcp.AddTool(s, &mcp.Tool{Name: "list_issues", Description: "List issues of a repository"}, makeRepoSublist(token, "issues"))
	mcp.AddTool(s, &mcp.Tool{Name: "get_issue", Description: "Get issue details"}, makeGetIssueOrPR(token, "issues"))
	mcp.AddTool(s, &mcp.Tool{Name: "list_issue_comments", Description: "List comments on an issue"}, makeIssueComments(token))

	// Pull Requests
	mcp.AddTool(s, &mcp.Tool{Name: "list_pull_requests", Description: "List pull requests of a repository"}, makeRepoSublist(token, "pulls"))
	mcp.AddTool(s, &mcp.Tool{Name: "get_pull_request", Description: "Get pull request details"}, makeGetIssueOrPR(token, "pulls"))
	mcp.AddTool(s, &mcp.Tool{Name: "list_pr_files", Description: "List files changed in a pull request"}, makePRFiles(token))

	// Actions
	mcp.AddTool(s, &mcp.Tool{Name: "list_workflow_runs", Description: "List workflow runs for a repository"}, makeRepoSublist(token, "actions/runs"))
	mcp.AddTool(s, &mcp.Tool{Name: "get_workflow_run", Description: "Get a workflow run by ID"}, makeGetWorkflowRun(token))

	// Releases
	mcp.AddTool(s, &mcp.Tool{Name: "list_releases", Description: "List releases of a repository"}, makeRepoSublist(token, "releases"))
	mcp.AddTool(s, &mcp.Tool{Name: "get_latest_release", Description: "Get the latest release of a repository"}, makeLatestRelease(token))

	// Code Search
	mcp.AddTool(s, &mcp.Tool{Name: "search_code", Description: "Search code across repositories"}, makeSearchCode(token))
	mcp.AddTool(s, &mcp.Tool{Name: "search_repos", Description: "Search repositories"}, makeSearchRepos(token))
	mcp.AddTool(s, &mcp.Tool{Name: "search_issues", Description: "Search issues and pull requests"}, makeSearchIssues(token))

	// Users / Orgs
	mcp.AddTool(s, &mcp.Tool{Name: "get_user", Description: "Get user profile"}, makeGetUser(token))
	mcp.AddTool(s, &mcp.Tool{Name: "list_org_members", Description: "List organization members"}, makeOrgMembers(token))

	srv.ListenAndServe(config.Get("PORT", ":8000"))
}

// --- Param Types ---

type OwnerParams struct {
	Owner   string `json:"owner" jsonschema:"Repository owner (user or org)"`
	PerPage int    `json:"per_page,omitempty" jsonschema:"Items per page, max 100 (default 30)"`
	Page    int    `json:"page,omitempty" jsonschema:"Page number"`
}

type RepoParams struct {
	Owner string `json:"owner" jsonschema:"Repository owner"`
	Repo  string `json:"repo" jsonschema:"Repository name"`
}

type RepoListParams struct {
	Owner   string `json:"owner" jsonschema:"Repository owner"`
	Repo    string `json:"repo" jsonschema:"Repository name"`
	PerPage int    `json:"per_page,omitempty" jsonschema:"Items per page, max 100 (default 30)"`
	Page    int    `json:"page,omitempty" jsonschema:"Page number"`
	State   string `json:"state,omitempty" jsonschema:"Filter by state: open, closed, all (default open)"`
}

type IssueParams struct {
	Owner  string `json:"owner" jsonschema:"Repository owner"`
	Repo   string `json:"repo" jsonschema:"Repository name"`
	Number int    `json:"number" jsonschema:"Issue or PR number"`
}

type ContentParams struct {
	Owner string `json:"owner" jsonschema:"Repository owner"`
	Repo  string `json:"repo" jsonschema:"Repository name"`
	Path  string `json:"path" jsonschema:"File or directory path"`
	Ref   string `json:"ref,omitempty" jsonschema:"Branch, tag, or commit SHA (default: default branch)"`
}

type SearchParams struct {
	Query   string `json:"query" jsonschema:"Search query using GitHub search syntax"`
	PerPage int    `json:"per_page,omitempty" jsonschema:"Results per page, max 100 (default 30)"`
	Page    int    `json:"page,omitempty" jsonschema:"Page number"`
}

type UserParams struct {
	Username string `json:"username" jsonschema:"GitHub username"`
}

type OrgParams struct {
	Org     string `json:"org" jsonschema:"Organization name"`
	PerPage int    `json:"per_page,omitempty" jsonschema:"Items per page"`
	Page    int    `json:"page,omitempty" jsonschema:"Page number"`
}

type RunParams struct {
	Owner string `json:"owner" jsonschema:"Repository owner"`
	Repo  string `json:"repo" jsonschema:"Repository name"`
	RunID int64  `json:"run_id" jsonschema:"Workflow run ID"`
}

// --- Handlers ---

func makeListRepos(token string) func(context.Context, *mcp.CallToolRequest, *OwnerParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *OwnerParams) (*mcp.CallToolResult, any, error) {
		if p.Owner == "" {
			return errResult("owner is required")
		}
		qp := pagination(p.PerPage, p.Page)
		return ghGet(token, fmt.Sprintf("/users/%s/repos", p.Owner), qp)
	}
}

func makeGetRepo(token string) func(context.Context, *mcp.CallToolRequest, *RepoParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *RepoParams) (*mcp.CallToolResult, any, error) {
		if p.Owner == "" || p.Repo == "" {
			return errResult("owner and repo are required")
		}
		return ghGet(token, fmt.Sprintf("/repos/%s/%s", p.Owner, p.Repo), nil)
	}
}

func makeRepoSublist(token, sub string) func(context.Context, *mcp.CallToolRequest, *RepoListParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *RepoListParams) (*mcp.CallToolResult, any, error) {
		if p.Owner == "" || p.Repo == "" {
			return errResult("owner and repo are required")
		}
		qp := pagination(p.PerPage, p.Page)
		if p.State != "" {
			qp["state"] = p.State
		}
		return ghGet(token, fmt.Sprintf("/repos/%s/%s/%s", p.Owner, p.Repo, sub), qp)
	}
}

func makeGetIssueOrPR(token, kind string) func(context.Context, *mcp.CallToolRequest, *IssueParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IssueParams) (*mcp.CallToolResult, any, error) {
		if p.Owner == "" || p.Repo == "" || p.Number <= 0 {
			return errResult("owner, repo, and number are required")
		}
		return ghGet(token, fmt.Sprintf("/repos/%s/%s/%s/%d", p.Owner, p.Repo, kind, p.Number), nil)
	}
}

func makeIssueComments(token string) func(context.Context, *mcp.CallToolRequest, *IssueParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IssueParams) (*mcp.CallToolResult, any, error) {
		if p.Owner == "" || p.Repo == "" || p.Number <= 0 {
			return errResult("owner, repo, and number are required")
		}
		return ghGet(token, fmt.Sprintf("/repos/%s/%s/issues/%d/comments", p.Owner, p.Repo, p.Number), nil)
	}
}

func makePRFiles(token string) func(context.Context, *mcp.CallToolRequest, *IssueParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IssueParams) (*mcp.CallToolResult, any, error) {
		if p.Owner == "" || p.Repo == "" || p.Number <= 0 {
			return errResult("owner, repo, and number are required")
		}
		return ghGet(token, fmt.Sprintf("/repos/%s/%s/pulls/%d/files", p.Owner, p.Repo, p.Number), nil)
	}
}

func makeGetContent(token string) func(context.Context, *mcp.CallToolRequest, *ContentParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ContentParams) (*mcp.CallToolResult, any, error) {
		if p.Owner == "" || p.Repo == "" || p.Path == "" {
			return errResult("owner, repo, and path are required")
		}
		qp := map[string]string{}
		if p.Ref != "" {
			qp["ref"] = p.Ref
		}
		return ghGet(token, fmt.Sprintf("/repos/%s/%s/contents/%s", p.Owner, p.Repo, p.Path), qp)
	}
}

func makeGetWorkflowRun(token string) func(context.Context, *mcp.CallToolRequest, *RunParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *RunParams) (*mcp.CallToolResult, any, error) {
		if p.Owner == "" || p.Repo == "" || p.RunID <= 0 {
			return errResult("owner, repo, and run_id are required")
		}
		return ghGet(token, fmt.Sprintf("/repos/%s/%s/actions/runs/%d", p.Owner, p.Repo, p.RunID), nil)
	}
}

func makeLatestRelease(token string) func(context.Context, *mcp.CallToolRequest, *RepoParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *RepoParams) (*mcp.CallToolResult, any, error) {
		if p.Owner == "" || p.Repo == "" {
			return errResult("owner and repo are required")
		}
		return ghGet(token, fmt.Sprintf("/repos/%s/%s/releases/latest", p.Owner, p.Repo), nil)
	}
}

func makeSearchCode(token string) func(context.Context, *mcp.CallToolRequest, *SearchParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *SearchParams) (*mcp.CallToolResult, any, error) {
		if p.Query == "" {
			return errResult("query is required")
		}
		qp := map[string]string{"q": p.Query}
		addPagination(qp, p.PerPage, p.Page)
		return ghGet(token, "/search/code", qp)
	}
}

func makeSearchRepos(token string) func(context.Context, *mcp.CallToolRequest, *SearchParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *SearchParams) (*mcp.CallToolResult, any, error) {
		if p.Query == "" {
			return errResult("query is required")
		}
		qp := map[string]string{"q": p.Query}
		addPagination(qp, p.PerPage, p.Page)
		return ghGet(token, "/search/repositories", qp)
	}
}

func makeSearchIssues(token string) func(context.Context, *mcp.CallToolRequest, *SearchParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *SearchParams) (*mcp.CallToolResult, any, error) {
		if p.Query == "" {
			return errResult("query is required")
		}
		qp := map[string]string{"q": p.Query}
		addPagination(qp, p.PerPage, p.Page)
		return ghGet(token, "/search/issues", qp)
	}
}

func makeGetUser(token string) func(context.Context, *mcp.CallToolRequest, *UserParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *UserParams) (*mcp.CallToolResult, any, error) {
		if p.Username == "" {
			return errResult("username is required")
		}
		return ghGet(token, fmt.Sprintf("/users/%s", p.Username), nil)
	}
}

func makeOrgMembers(token string) func(context.Context, *mcp.CallToolRequest, *OrgParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *OrgParams) (*mcp.CallToolResult, any, error) {
		if p.Org == "" {
			return errResult("org is required")
		}
		qp := pagination(p.PerPage, p.Page)
		return ghGet(token, fmt.Sprintf("/orgs/%s/members", p.Org), qp)
	}
}

// --- API Client ---

func ghGet(token, path string, params map[string]string) (*mcp.CallToolResult, any, error) {
	url := baseURL + path
	if !strings.HasPrefix(path, "/") {
		url = baseURL + "/" + path
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return errResult(err.Error())
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

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
		return errResult(fmt.Sprintf("GitHub API error %d: %s", resp.StatusCode, string(body)))
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

func pagination(perPage, page int) map[string]string {
	qp := map[string]string{}
	addPagination(qp, perPage, page)
	return qp
}

func addPagination(qp map[string]string, perPage, page int) {
	if perPage > 0 {
		qp["per_page"] = fmt.Sprintf("%d", perPage)
	}
	if page > 0 {
		qp["page"] = fmt.Sprintf("%d", page)
	}
}

func errResult(msg string) (*mcp.CallToolResult, any, error) {
	r := &mcp.CallToolResult{}
	r.SetError(fmt.Errorf("%s", msg))
	return r, nil, nil
}
