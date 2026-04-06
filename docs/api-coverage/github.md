# GitHub REST API Coverage

- **API**: [GitHub REST API](https://docs.github.com/en/rest)
- **API Version**: 2022-11-28
- **Letzter Check**: 2026-04-06
- **Scope**: readonly

## Endpoints

| Bereich | Endpoint | Status | Tool-Name |
|---------|----------|--------|-----------|
| Repos | GET /users/{user}/repos | implemented | list_repos |
| Repos | GET /repos/{owner}/{repo} | implemented | get_repo |
| Repos | GET /repos/{owner}/{repo}/branches | implemented | list_branches |
| Repos | GET /repos/{owner}/{repo}/commits | implemented | list_commits |
| Repos | GET /repos/{owner}/{repo}/contents/{path} | implemented | get_repo_content |
| Repos | POST/PUT/DELETE | out-of-scope | - |
| Issues | GET /repos/{owner}/{repo}/issues | implemented | list_issues |
| Issues | GET /repos/{owner}/{repo}/issues/{number} | implemented | get_issue |
| Issues | GET /repos/{owner}/{repo}/issues/{number}/comments | implemented | list_issue_comments |
| Issues | POST (create/comment) | out-of-scope | - |
| Pull Requests | GET /repos/{owner}/{repo}/pulls | implemented | list_pull_requests |
| Pull Requests | GET /repos/{owner}/{repo}/pulls/{number} | implemented | get_pull_request |
| Pull Requests | GET /repos/{owner}/{repo}/pulls/{number}/files | implemented | list_pr_files |
| Pull Requests | POST (create/merge/review) | out-of-scope | - |
| Actions | GET /repos/{owner}/{repo}/actions/runs | implemented | list_workflow_runs |
| Actions | GET /repos/{owner}/{repo}/actions/runs/{id} | implemented | get_workflow_run |
| Actions | POST (trigger/cancel) | out-of-scope | - |
| Releases | GET /repos/{owner}/{repo}/releases | implemented | list_releases |
| Releases | GET /repos/{owner}/{repo}/releases/latest | implemented | get_latest_release |
| Releases | POST (create) | out-of-scope | - |
| Search | GET /search/code | implemented | search_code |
| Search | GET /search/repositories | implemented | search_repos |
| Search | GET /search/issues | implemented | search_issues |
| Users | GET /users/{username} | implemented | get_user |
| Orgs | GET /orgs/{org}/members | implemented | list_org_members |
| Notifications | GET /notifications | out-of-scope | - |
| Gists | GET /gists | out-of-scope | - |
| Projects | GET /projects | out-of-scope | - |

## Hinweise

- Alle schreibenden Endpoints sind out-of-scope (readonly)
- Auth: `Authorization: Bearer` + `X-GitHub-Api-Version: 2022-11-28`
- Pagination: `per_page` (max 100), `page`
- State-Filter fuer Issues/PRs: `state=open|closed|all`
