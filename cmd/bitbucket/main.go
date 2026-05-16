package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/traffino/mcp-server/internal/config"
	"github.com/traffino/mcp-server/internal/server"
)

const baseURL = "https://api.bitbucket.org/2.0"

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
	bbEmail    string
)

func main() {
	bbEmail = config.Require("BITBUCKET_USER_EMAIL")
	token := config.Require("BITBUCKET_API_TOKEN")
	srv := server.New("bitbucket", "1.0.0")
	s := srv.MCPServer()

	mcp.AddTool(s, &mcp.Tool{Name: "get_me", Description: "Get authenticated user"}, makeGetMe(token))

	mcp.AddTool(s, &mcp.Tool{Name: "list_workspaces", Description: "List workspaces the user belongs to"}, makeListWorkspaces(token))
	mcp.AddTool(s, &mcp.Tool{Name: "list_repositories", Description: "List repositories in a workspace"}, makeListRepositories(token))
	mcp.AddTool(s, &mcp.Tool{Name: "get_repository", Description: "Get repository details"}, makeGetRepository(token))
	mcp.AddTool(s, &mcp.Tool{Name: "get_file_contents", Description: "Get file contents at a specific commit/branch"}, makeGetFileContents(token))
	mcp.AddTool(s, &mcp.Tool{Name: "list_branches", Description: "List branches of a repository"}, makeListBranches(token))
	mcp.AddTool(s, &mcp.Tool{Name: "list_commits", Description: "List commits in a repository (optionally filtered by branch/tag)"}, makeListCommits(token))
	mcp.AddTool(s, &mcp.Tool{Name: "get_commit", Description: "Get a single commit by hash"}, makeGetCommit(token))

	mcp.AddTool(s, &mcp.Tool{Name: "list_pull_requests", Description: "List pull requests (filter by state: OPEN, MERGED, DECLINED, SUPERSEDED)"}, makeListPullRequests(token))
	mcp.AddTool(s, &mcp.Tool{Name: "pull_request_read", Description: "Get pull request details, optionally including diff/activity/commits/statuses"}, makePullRequestRead(token))
	mcp.AddTool(s, &mcp.Tool{Name: "pull_request_write", Description: "Create or update a pull request (action: create|update)"}, makePullRequestWrite(token))
	mcp.AddTool(s, &mcp.Tool{Name: "pull_request_review_write", Description: "Approve, unapprove, request-changes, unrequest-changes or decline a pull request"}, makePullRequestReviewWrite(token))
	mcp.AddTool(s, &mcp.Tool{Name: "pull_request_comment_write", Description: "Post a general or inline comment on a pull request"}, makePullRequestCommentWrite(token))
	mcp.AddTool(s, &mcp.Tool{Name: "merge_pull_request", Description: "Merge a pull request (merge_strategy: merge_commit|squash|fast_forward)"}, makeMergePullRequest(token))

	mcp.AddTool(s, &mcp.Tool{Name: "list_pipelines", Description: "List pipelines in a repository"}, makeListPipelines(token))
	mcp.AddTool(s, &mcp.Tool{Name: "get_pipeline", Description: "Get pipeline details, optionally with steps"}, makeGetPipeline(token))

	mcp.AddTool(s, &mcp.Tool{Name: "list_issues", Description: "List repository issues (requires Issues enabled on the repo)"}, makeListIssues(token))
	mcp.AddTool(s, &mcp.Tool{Name: "issue_read", Description: "Get a single issue by ID, optionally with comments"}, makeIssueRead(token))
	mcp.AddTool(s, &mcp.Tool{Name: "issue_write", Description: "Create, update or comment on an issue (action: create|update|comment)"}, makeIssueWrite(token))

	srv.ListenAndServe(config.Get("PORT", ":8000"))
}

// --- Param Types ---

type RepoParams struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug (e.g. baltasaar)"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
}

type ListWorkspacesParams struct {
	PageLen int `json:"pagelen,omitempty" jsonschema:"Results per page, 1-100 (default 50)"`
	Page    int `json:"page,omitempty" jsonschema:"Page number (1-indexed)"`
}

type ListRepositoriesParams struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	Query     string `json:"q,omitempty" jsonschema:"BBQL query, e.g. 'name ~ \"my-repo\"'"`
	Sort      string `json:"sort,omitempty" jsonschema:"Sort field, e.g. -updated_on, name"`
	PageLen   int    `json:"pagelen,omitempty" jsonschema:"Results per page, 1-100 (default 50)"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number (1-indexed)"`
}

type GetFileContentsParams struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Path      string `json:"path" jsonschema:"Path inside the repository"`
	Ref       string `json:"ref,omitempty" jsonschema:"Commit hash, branch or tag (default: repository main branch)"`
}

type ListCommitsParams struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Include   string `json:"include,omitempty" jsonschema:"Branch, tag or commit hash to include"`
	Exclude   string `json:"exclude,omitempty" jsonschema:"Branch, tag or commit hash to exclude"`
	PageLen   int    `json:"pagelen,omitempty" jsonschema:"Results per page, 1-100 (default 30)"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number (1-indexed)"`
}

type GetCommitParams struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Hash      string `json:"hash" jsonschema:"Commit hash (full or short)"`
}

type ListBranchesParams struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Query     string `json:"q,omitempty" jsonschema:"BBQL query, e.g. 'name ~ \"feature/\"'"`
	PageLen   int    `json:"pagelen,omitempty" jsonschema:"Results per page, 1-100 (default 30)"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number (1-indexed)"`
}

type ListPullRequestsParams struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	State     string `json:"state,omitempty" jsonschema:"OPEN, MERGED, DECLINED, SUPERSEDED (default OPEN)"`
	Query     string `json:"q,omitempty" jsonschema:"BBQL query, e.g. 'source.branch.name = \"feature/x\"'"`
	PageLen   int    `json:"pagelen,omitempty" jsonschema:"Results per page, 1-50 (default 25)"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number (1-indexed)"`
}

type PullRequestReadParams struct {
	Workspace     string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug      string `json:"repo_slug" jsonschema:"Repository slug"`
	PullRequestID int    `json:"pull_request_id" jsonschema:"PR number"`
	Include       string `json:"include,omitempty" jsonschema:"Optional extra: diff, activity, commits, statuses"`
}

type PullRequestWriteParams struct {
	Action            string   `json:"action" jsonschema:"create or update"`
	Workspace         string   `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug          string   `json:"repo_slug" jsonschema:"Repository slug"`
	PullRequestID     int      `json:"pull_request_id,omitempty" jsonschema:"PR number (required for update)"`
	Title             string   `json:"title,omitempty" jsonschema:"PR title (required for create)"`
	Description       string   `json:"description,omitempty" jsonschema:"PR description (markdown)"`
	SourceBranch      string   `json:"source_branch,omitempty" jsonschema:"Source branch name (required for create)"`
	DestinationBranch string   `json:"destination_branch,omitempty" jsonschema:"Destination branch (default: repository main branch)"`
	CloseSourceBranch bool     `json:"close_source_branch,omitempty" jsonschema:"Delete source branch after merge"`
	Reviewers         []string `json:"reviewers,omitempty" jsonschema:"List of reviewer UUIDs"`
}

type PullRequestReviewParams struct {
	Action        string `json:"action" jsonschema:"approve, unapprove, request-changes, unrequest-changes, decline"`
	Workspace     string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug      string `json:"repo_slug" jsonschema:"Repository slug"`
	PullRequestID int    `json:"pull_request_id" jsonschema:"PR number"`
	Message       string `json:"message,omitempty" jsonschema:"Decline reason (only for action=decline)"`
}

type PullRequestCommentParams struct {
	Workspace     string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug      string `json:"repo_slug" jsonschema:"Repository slug"`
	PullRequestID int    `json:"pull_request_id" jsonschema:"PR number"`
	Content       string `json:"content" jsonschema:"Comment markdown"`
	InlinePath    string `json:"inline_path,omitempty" jsonschema:"File path for inline comment"`
	InlineLine    int    `json:"inline_line,omitempty" jsonschema:"Line number in destination/new file for inline comment"`
}

type MergePullRequestParams struct {
	Workspace         string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug          string `json:"repo_slug" jsonschema:"Repository slug"`
	PullRequestID     int    `json:"pull_request_id" jsonschema:"PR number"`
	MergeStrategy     string `json:"merge_strategy,omitempty" jsonschema:"merge_commit, squash, fast_forward (default: repo setting)"`
	Message           string `json:"message,omitempty" jsonschema:"Merge commit message"`
	CloseSourceBranch bool   `json:"close_source_branch,omitempty" jsonschema:"Delete source branch after merge"`
}

type ListPipelinesParams struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Sort      string `json:"sort,omitempty" jsonschema:"Sort field, e.g. -created_on"`
	PageLen   int    `json:"pagelen,omitempty" jsonschema:"Results per page (default 30)"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number (1-indexed)"`
}

type GetPipelineParams struct {
	Workspace    string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug     string `json:"repo_slug" jsonschema:"Repository slug"`
	PipelineUUID string `json:"pipeline_uuid" jsonschema:"Pipeline UUID (with or without braces)"`
	IncludeSteps bool   `json:"include_steps,omitempty" jsonschema:"Also fetch pipeline steps"`
}

type ListIssuesParams struct {
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	Query     string `json:"q,omitempty" jsonschema:"BBQL query, e.g. 'state = \"open\"'"`
	Sort      string `json:"sort,omitempty" jsonschema:"Sort field, e.g. -updated_on"`
	PageLen   int    `json:"pagelen,omitempty" jsonschema:"Results per page (default 30)"`
	Page      int    `json:"page,omitempty" jsonschema:"Page number (1-indexed)"`
}

type IssueReadParams struct {
	Workspace       string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug        string `json:"repo_slug" jsonschema:"Repository slug"`
	IssueID         int    `json:"issue_id" jsonschema:"Issue number"`
	IncludeComments bool   `json:"include_comments,omitempty" jsonschema:"Also fetch comments"`
}

type IssueWriteParams struct {
	Action    string `json:"action" jsonschema:"create, update or comment"`
	Workspace string `json:"workspace" jsonschema:"Workspace slug"`
	RepoSlug  string `json:"repo_slug" jsonschema:"Repository slug"`
	IssueID   int    `json:"issue_id,omitempty" jsonschema:"Issue number (required for update/comment)"`
	Title     string `json:"title,omitempty" jsonschema:"Issue title (required for create)"`
	Content   string `json:"content,omitempty" jsonschema:"Issue body markdown (create/update) or comment text (comment)"`
	Kind      string `json:"kind,omitempty" jsonschema:"bug, enhancement, proposal, task (default bug)"`
	Priority  string `json:"priority,omitempty" jsonschema:"trivial, minor, major, critical, blocker"`
	State     string `json:"state,omitempty" jsonschema:"new, open, resolved, on hold, invalid, duplicate, wontfix, closed"`
	Assignee  string `json:"assignee,omitempty" jsonschema:"Assignee account ID or UUID"`
}

// --- Handlers ---

func makeGetMe(token string) func(context.Context, *mcp.CallToolRequest, any) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
		return bbGet(token, "/user", nil)
	}
}

func makeListWorkspaces(token string) func(context.Context, *mcp.CallToolRequest, *ListWorkspacesParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ListWorkspacesParams) (*mcp.CallToolResult, any, error) {
		q := url.Values{}
		setPagination(q, p.Page, p.PageLen, 50)
		return bbGet(token, "/workspaces", q)
	}
}

func makeListRepositories(token string) func(context.Context, *mcp.CallToolRequest, *ListRepositoriesParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ListRepositoriesParams) (*mcp.CallToolResult, any, error) {
		if p.Workspace == "" {
			return errResult("workspace is required")
		}
		q := url.Values{}
		if p.Query != "" {
			q.Set("q", p.Query)
		}
		if p.Sort != "" {
			q.Set("sort", p.Sort)
		}
		setPagination(q, p.Page, p.PageLen, 50)
		return bbGet(token, fmt.Sprintf("/repositories/%s", p.Workspace), q)
	}
}

func makeGetRepository(token string) func(context.Context, *mcp.CallToolRequest, *RepoParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *RepoParams) (*mcp.CallToolResult, any, error) {
		if p.Workspace == "" || p.RepoSlug == "" {
			return errResult("workspace and repo_slug are required")
		}
		return bbGet(token, fmt.Sprintf("/repositories/%s/%s", p.Workspace, p.RepoSlug), nil)
	}
}

func makeGetFileContents(token string) func(context.Context, *mcp.CallToolRequest, *GetFileContentsParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *GetFileContentsParams) (*mcp.CallToolResult, any, error) {
		if p.Workspace == "" || p.RepoSlug == "" || p.Path == "" {
			return errResult("workspace, repo_slug and path are required")
		}
		ref := p.Ref
		if ref == "" {
			repoBody, err := bbGetRaw(token, fmt.Sprintf("/repositories/%s/%s", p.Workspace, p.RepoSlug), nil)
			if err != nil {
				return errResult(err.Error())
			}
			var r struct {
				Mainbranch struct {
					Name string `json:"name"`
				} `json:"mainbranch"`
			}
			if err := json.Unmarshal(repoBody, &r); err != nil || r.Mainbranch.Name == "" {
				return errResult("could not determine default branch; pass ref explicitly")
			}
			ref = r.Mainbranch.Name
		}
		return bbGetText(token, fmt.Sprintf("/repositories/%s/%s/src/%s/%s", p.Workspace, p.RepoSlug, url.PathEscape(ref), p.Path))
	}
}

func makeListBranches(token string) func(context.Context, *mcp.CallToolRequest, *ListBranchesParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ListBranchesParams) (*mcp.CallToolResult, any, error) {
		if p.Workspace == "" || p.RepoSlug == "" {
			return errResult("workspace and repo_slug are required")
		}
		q := url.Values{}
		if p.Query != "" {
			q.Set("q", p.Query)
		}
		setPagination(q, p.Page, p.PageLen, 30)
		return bbGet(token, fmt.Sprintf("/repositories/%s/%s/refs/branches", p.Workspace, p.RepoSlug), q)
	}
}

func makeListCommits(token string) func(context.Context, *mcp.CallToolRequest, *ListCommitsParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ListCommitsParams) (*mcp.CallToolResult, any, error) {
		if p.Workspace == "" || p.RepoSlug == "" {
			return errResult("workspace and repo_slug are required")
		}
		q := url.Values{}
		if p.Include != "" {
			q.Set("include", p.Include)
		}
		if p.Exclude != "" {
			q.Set("exclude", p.Exclude)
		}
		setPagination(q, p.Page, p.PageLen, 30)
		return bbGet(token, fmt.Sprintf("/repositories/%s/%s/commits", p.Workspace, p.RepoSlug), q)
	}
}

func makeGetCommit(token string) func(context.Context, *mcp.CallToolRequest, *GetCommitParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *GetCommitParams) (*mcp.CallToolResult, any, error) {
		if p.Workspace == "" || p.RepoSlug == "" || p.Hash == "" {
			return errResult("workspace, repo_slug and hash are required")
		}
		return bbGet(token, fmt.Sprintf("/repositories/%s/%s/commit/%s", p.Workspace, p.RepoSlug, p.Hash), nil)
	}
}

func makeListPullRequests(token string) func(context.Context, *mcp.CallToolRequest, *ListPullRequestsParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ListPullRequestsParams) (*mcp.CallToolResult, any, error) {
		if p.Workspace == "" || p.RepoSlug == "" {
			return errResult("workspace and repo_slug are required")
		}
		q := url.Values{}
		state := p.State
		if state == "" {
			state = "OPEN"
		}
		q.Set("state", state)
		if p.Query != "" {
			q.Set("q", p.Query)
		}
		setPagination(q, p.Page, p.PageLen, 25)
		return bbGet(token, fmt.Sprintf("/repositories/%s/%s/pullrequests", p.Workspace, p.RepoSlug), q)
	}
}

func makePullRequestRead(token string) func(context.Context, *mcp.CallToolRequest, *PullRequestReadParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *PullRequestReadParams) (*mcp.CallToolResult, any, error) {
		if p.Workspace == "" || p.RepoSlug == "" || p.PullRequestID == 0 {
			return errResult("workspace, repo_slug and pull_request_id are required")
		}
		path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d", p.Workspace, p.RepoSlug, p.PullRequestID)
		switch p.Include {
		case "":
			return bbGet(token, path, nil)
		case "diff":
			return bbGetText(token, path+"/diff")
		case "activity", "commits", "statuses":
			return bbGet(token, path+"/"+p.Include, nil)
		default:
			return errResult("include must be empty, diff, activity, commits, or statuses")
		}
	}
}

func makePullRequestWrite(token string) func(context.Context, *mcp.CallToolRequest, *PullRequestWriteParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *PullRequestWriteParams) (*mcp.CallToolResult, any, error) {
		if p.Workspace == "" || p.RepoSlug == "" {
			return errResult("workspace and repo_slug are required")
		}
		switch p.Action {
		case "create":
			if p.Title == "" || p.SourceBranch == "" {
				return errResult("title and source_branch are required for create")
			}
			body := map[string]any{
				"title":  p.Title,
				"source": map[string]any{"branch": map[string]any{"name": p.SourceBranch}},
			}
			if p.DestinationBranch != "" {
				body["destination"] = map[string]any{"branch": map[string]any{"name": p.DestinationBranch}}
			}
			if p.Description != "" {
				body["description"] = p.Description
			}
			if p.CloseSourceBranch {
				body["close_source_branch"] = true
			}
			if len(p.Reviewers) > 0 {
				revs := make([]map[string]string, 0, len(p.Reviewers))
				for _, r := range p.Reviewers {
					revs = append(revs, map[string]string{"uuid": r})
				}
				body["reviewers"] = revs
			}
			return bbJSON(token, "POST", fmt.Sprintf("/repositories/%s/%s/pullrequests", p.Workspace, p.RepoSlug), body)
		case "update":
			if p.PullRequestID == 0 {
				return errResult("pull_request_id is required for update")
			}
			body := map[string]any{}
			if p.Title != "" {
				body["title"] = p.Title
			}
			if p.Description != "" {
				body["description"] = p.Description
			}
			if p.DestinationBranch != "" {
				body["destination"] = map[string]any{"branch": map[string]any{"name": p.DestinationBranch}}
			}
			if len(body) == 0 {
				return errResult("update requires at least one of title, description, destination_branch")
			}
			return bbJSON(token, "PUT", fmt.Sprintf("/repositories/%s/%s/pullrequests/%d", p.Workspace, p.RepoSlug, p.PullRequestID), body)
		default:
			return errResult("action must be create or update")
		}
	}
}

func makePullRequestReviewWrite(token string) func(context.Context, *mcp.CallToolRequest, *PullRequestReviewParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *PullRequestReviewParams) (*mcp.CallToolResult, any, error) {
		if p.Workspace == "" || p.RepoSlug == "" || p.PullRequestID == 0 {
			return errResult("workspace, repo_slug and pull_request_id are required")
		}
		path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d", p.Workspace, p.RepoSlug, p.PullRequestID)
		switch p.Action {
		case "approve":
			return bbJSON(token, "POST", path+"/approve", nil)
		case "unapprove":
			return bbJSON(token, "DELETE", path+"/approve", nil)
		case "request-changes":
			return bbJSON(token, "POST", path+"/request-changes", nil)
		case "unrequest-changes":
			return bbJSON(token, "DELETE", path+"/request-changes", nil)
		case "decline":
			var body any
			if p.Message != "" {
				body = map[string]any{"message": p.Message}
			}
			return bbJSON(token, "POST", path+"/decline", body)
		default:
			return errResult("action must be approve, unapprove, request-changes, unrequest-changes or decline")
		}
	}
}

func makePullRequestCommentWrite(token string) func(context.Context, *mcp.CallToolRequest, *PullRequestCommentParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *PullRequestCommentParams) (*mcp.CallToolResult, any, error) {
		if p.Workspace == "" || p.RepoSlug == "" || p.PullRequestID == 0 || p.Content == "" {
			return errResult("workspace, repo_slug, pull_request_id and content are required")
		}
		body := map[string]any{"content": map[string]any{"raw": p.Content}}
		if p.InlinePath != "" {
			inline := map[string]any{"path": p.InlinePath}
			if p.InlineLine > 0 {
				inline["to"] = p.InlineLine
			}
			body["inline"] = inline
		}
		return bbJSON(token, "POST", fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments", p.Workspace, p.RepoSlug, p.PullRequestID), body)
	}
}

func makeMergePullRequest(token string) func(context.Context, *mcp.CallToolRequest, *MergePullRequestParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *MergePullRequestParams) (*mcp.CallToolResult, any, error) {
		if p.Workspace == "" || p.RepoSlug == "" || p.PullRequestID == 0 {
			return errResult("workspace, repo_slug and pull_request_id are required")
		}
		body := map[string]any{}
		if p.MergeStrategy != "" {
			body["merge_strategy"] = p.MergeStrategy
		}
		if p.Message != "" {
			body["message"] = p.Message
		}
		if p.CloseSourceBranch {
			body["close_source_branch"] = true
		}
		return bbJSON(token, "POST", fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/merge", p.Workspace, p.RepoSlug, p.PullRequestID), body)
	}
}

func makeListPipelines(token string) func(context.Context, *mcp.CallToolRequest, *ListPipelinesParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ListPipelinesParams) (*mcp.CallToolResult, any, error) {
		if p.Workspace == "" || p.RepoSlug == "" {
			return errResult("workspace and repo_slug are required")
		}
		q := url.Values{}
		if p.Sort != "" {
			q.Set("sort", p.Sort)
		}
		setPagination(q, p.Page, p.PageLen, 30)
		return bbGet(token, fmt.Sprintf("/repositories/%s/%s/pipelines/", p.Workspace, p.RepoSlug), q)
	}
}

func makeGetPipeline(token string) func(context.Context, *mcp.CallToolRequest, *GetPipelineParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *GetPipelineParams) (*mcp.CallToolResult, any, error) {
		if p.Workspace == "" || p.RepoSlug == "" || p.PipelineUUID == "" {
			return errResult("workspace, repo_slug and pipeline_uuid are required")
		}
		uuid := p.PipelineUUID
		if len(uuid) > 0 && uuid[0] != '{' {
			uuid = "{" + uuid + "}"
		}
		base := fmt.Sprintf("/repositories/%s/%s/pipelines/%s", p.Workspace, p.RepoSlug, uuid)
		if !p.IncludeSteps {
			return bbGet(token, base, nil)
		}
		pipeline, err := bbGetRaw(token, base, nil)
		if err != nil {
			return errResult(err.Error())
		}
		steps, err := bbGetRaw(token, base+"/steps/", nil)
		if err != nil {
			return errResult(err.Error())
		}
		out := map[string]json.RawMessage{
			"pipeline": pipeline,
			"steps":    steps,
		}
		return jsonResult(out)
	}
}

func makeListIssues(token string) func(context.Context, *mcp.CallToolRequest, *ListIssuesParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *ListIssuesParams) (*mcp.CallToolResult, any, error) {
		if p.Workspace == "" || p.RepoSlug == "" {
			return errResult("workspace and repo_slug are required")
		}
		q := url.Values{}
		if p.Query != "" {
			q.Set("q", p.Query)
		}
		if p.Sort != "" {
			q.Set("sort", p.Sort)
		}
		setPagination(q, p.Page, p.PageLen, 30)
		return bbGet(token, fmt.Sprintf("/repositories/%s/%s/issues", p.Workspace, p.RepoSlug), q)
	}
}

func makeIssueRead(token string) func(context.Context, *mcp.CallToolRequest, *IssueReadParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IssueReadParams) (*mcp.CallToolResult, any, error) {
		if p.Workspace == "" || p.RepoSlug == "" || p.IssueID == 0 {
			return errResult("workspace, repo_slug and issue_id are required")
		}
		base := fmt.Sprintf("/repositories/%s/%s/issues/%d", p.Workspace, p.RepoSlug, p.IssueID)
		if !p.IncludeComments {
			return bbGet(token, base, nil)
		}
		issue, err := bbGetRaw(token, base, nil)
		if err != nil {
			return errResult(err.Error())
		}
		comments, err := bbGetRaw(token, base+"/comments", nil)
		if err != nil {
			return errResult(err.Error())
		}
		out := map[string]json.RawMessage{
			"issue":    issue,
			"comments": comments,
		}
		return jsonResult(out)
	}
}

func makeIssueWrite(token string) func(context.Context, *mcp.CallToolRequest, *IssueWriteParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, p *IssueWriteParams) (*mcp.CallToolResult, any, error) {
		if p.Workspace == "" || p.RepoSlug == "" {
			return errResult("workspace and repo_slug are required")
		}
		switch p.Action {
		case "create":
			if p.Title == "" {
				return errResult("title is required for create")
			}
			body := map[string]any{"title": p.Title}
			if p.Content != "" {
				body["content"] = map[string]any{"raw": p.Content}
			}
			if p.Kind != "" {
				body["kind"] = p.Kind
			}
			if p.Priority != "" {
				body["priority"] = p.Priority
			}
			if p.Assignee != "" {
				body["assignee"] = map[string]any{"account_id": p.Assignee}
			}
			return bbJSON(token, "POST", fmt.Sprintf("/repositories/%s/%s/issues", p.Workspace, p.RepoSlug), body)
		case "update":
			if p.IssueID == 0 {
				return errResult("issue_id is required for update")
			}
			body := map[string]any{}
			if p.Title != "" {
				body["title"] = p.Title
			}
			if p.Content != "" {
				body["content"] = map[string]any{"raw": p.Content}
			}
			if p.Kind != "" {
				body["kind"] = p.Kind
			}
			if p.Priority != "" {
				body["priority"] = p.Priority
			}
			if p.State != "" {
				body["state"] = p.State
			}
			if p.Assignee != "" {
				body["assignee"] = map[string]any{"account_id": p.Assignee}
			}
			if len(body) == 0 {
				return errResult("update requires at least one updatable field")
			}
			return bbJSON(token, "PUT", fmt.Sprintf("/repositories/%s/%s/issues/%d", p.Workspace, p.RepoSlug, p.IssueID), body)
		case "comment":
			if p.IssueID == 0 || p.Content == "" {
				return errResult("issue_id and content are required for comment")
			}
			body := map[string]any{"content": map[string]any{"raw": p.Content}}
			return bbJSON(token, "POST", fmt.Sprintf("/repositories/%s/%s/issues/%d/comments", p.Workspace, p.RepoSlug, p.IssueID), body)
		default:
			return errResult("action must be create, update or comment")
		}
	}
}

// --- API Client ---

func setPagination(q url.Values, page, pageLen, defaultLen int) {
	if pageLen <= 0 {
		pageLen = defaultLen
	}
	if pageLen > 100 {
		pageLen = 100
	}
	q.Set("pagelen", strconv.Itoa(pageLen))
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
}

func bbRequest(token, method, path string, query url.Values, body any) (*http.Response, []byte, error) {
	u := baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, u, bodyReader)
	if err != nil {
		return nil, nil, err
	}
	req.SetBasicAuth(bbEmail, token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, err
	}
	return resp, respBody, nil
}

func bbGetRaw(token, path string, query url.Values) ([]byte, error) {
	resp, body, err := bbRequest(token, "GET", path, query, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Bitbucket API error %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func bbGet(token, path string, query url.Values) (*mcp.CallToolResult, any, error) {
	body, err := bbGetRaw(token, path, query)
	if err != nil {
		return errResult(err.Error())
	}
	return prettyJSONResult(body)
}

func bbGetText(token, path string) (*mcp.CallToolResult, any, error) {
	resp, body, err := bbRequest(token, "GET", path, nil, nil)
	if err != nil {
		return errResult(err.Error())
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errResult(fmt.Sprintf("Bitbucket API error %d: %s", resp.StatusCode, string(body)))
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(body)}}}, nil, nil
}

func bbJSON(token, method, path string, body any) (*mcp.CallToolResult, any, error) {
	resp, respBody, err := bbRequest(token, method, path, nil, body)
	if err != nil {
		return errResult(err.Error())
	}
	if resp.StatusCode == http.StatusNoContent {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "OK"}}}, nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errResult(fmt.Sprintf("Bitbucket API error %d: %s", resp.StatusCode, string(respBody)))
	}
	if len(respBody) == 0 {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "OK"}}}, nil, nil
	}
	return prettyJSONResult(respBody)
}

func prettyJSONResult(body []byte) (*mcp.CallToolResult, any, error) {
	var raw json.RawMessage
	if json.Unmarshal(body, &raw) == nil {
		if indented, err := json.MarshalIndent(raw, "", "  "); err == nil {
			body = indented
		}
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(body)}}}, nil, nil
}

func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errResult(err.Error())
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(body)}}}, nil, nil
}

func errResult(msg string) (*mcp.CallToolResult, any, error) {
	r := &mcp.CallToolResult{}
	r.SetError(fmt.Errorf("%s", msg))
	return r, nil, nil
}
