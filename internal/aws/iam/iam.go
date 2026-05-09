// Package iam registers AWS IAM read-only tools on an MCP server.
package iam

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/traffino/mcp-server/internal/aws/awsutil"
)

// Register adds IAM tools to the MCP server.
func Register(s *mcp.Server, cfg aws.Config) {
	client := iam.NewFromConfig(cfg)

	// list_users
	type ListUsersParams struct {
		MaxItems int `json:"max_items,omitempty" jsonschema:"Maximum number of users to return (default 100)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_users",
		Description: "List IAM users in the account.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListUsersParams) (*mcp.CallToolResult, any, error) {
		limit := 100
		if p.MaxItems > 0 {
			limit = p.MaxItems
		}
		var users []any
		paginator := iam.NewListUsersPaginator(client, &iam.ListUsersInput{})
		for paginator.HasMorePages() && len(users) < limit {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("iam:ListUsers", err)
			}
			for _, u := range page.Users {
				users = append(users, u)
				if len(users) >= limit {
					break
				}
			}
		}
		return awsutil.JSONResult(users)
	})

	// get_user
	type GetUserParams struct {
		UserName string `json:"user_name" jsonschema:"The IAM user name"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_user",
		Description: "Get details for a specific IAM user.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *GetUserParams) (*mcp.CallToolResult, any, error) {
		if p.UserName == "" {
			return awsutil.ErrResult("user_name is required")
		}
		out, err := client.GetUser(ctx, &iam.GetUserInput{
			UserName: aws.String(p.UserName),
		})
		if err != nil {
			return awsutil.AWSErr("iam:GetUser", err)
		}
		return awsutil.JSONResult(out.User)
	})

	// list_roles
	type ListRolesParams struct {
		MaxItems int `json:"max_items,omitempty" jsonschema:"Maximum number of roles to return (default 100)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_roles",
		Description: "List IAM roles in the account.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListRolesParams) (*mcp.CallToolResult, any, error) {
		limit := 100
		if p.MaxItems > 0 {
			limit = p.MaxItems
		}
		var roles []any
		paginator := iam.NewListRolesPaginator(client, &iam.ListRolesInput{})
		for paginator.HasMorePages() && len(roles) < limit {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("iam:ListRoles", err)
			}
			for _, r := range page.Roles {
				roles = append(roles, r)
				if len(roles) >= limit {
					break
				}
			}
		}
		return awsutil.JSONResult(roles)
	})

	// get_role
	type GetRoleParams struct {
		RoleName string `json:"role_name" jsonschema:"The IAM role name"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_role",
		Description: "Get details for a specific IAM role.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *GetRoleParams) (*mcp.CallToolResult, any, error) {
		if p.RoleName == "" {
			return awsutil.ErrResult("role_name is required")
		}
		out, err := client.GetRole(ctx, &iam.GetRoleInput{
			RoleName: aws.String(p.RoleName),
		})
		if err != nil {
			return awsutil.AWSErr("iam:GetRole", err)
		}
		return awsutil.JSONResult(out.Role)
	})

	// list_policies
	type ListPoliciesParams struct {
		Scope    string `json:"scope,omitempty"     jsonschema:"Policy scope: Local AWS or All (default Local)"`
		MaxItems int    `json:"max_items,omitempty" jsonschema:"Maximum number of policies to return (default 100)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_policies",
		Description: "List IAM policies. Scope: Local (default), AWS, or All.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListPoliciesParams) (*mcp.CallToolResult, any, error) {
		limit := 100
		if p.MaxItems > 0 {
			limit = p.MaxItems
		}
		scope := iamtypes.PolicyScopeTypeLocal
		switch p.Scope {
		case "AWS":
			scope = iamtypes.PolicyScopeTypeAws
		case "All":
			scope = iamtypes.PolicyScopeTypeAll
		}
		var policies []any
		paginator := iam.NewListPoliciesPaginator(client, &iam.ListPoliciesInput{
			Scope: scope,
		})
		for paginator.HasMorePages() && len(policies) < limit {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("iam:ListPolicies", err)
			}
			for _, pol := range page.Policies {
				policies = append(policies, pol)
				if len(policies) >= limit {
					break
				}
			}
		}
		return awsutil.JSONResult(policies)
	})

	// list_groups
	type ListGroupsParams struct {
		MaxItems int `json:"max_items,omitempty" jsonschema:"Maximum number of groups to return (default 100)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_groups",
		Description: "List IAM groups in the account.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListGroupsParams) (*mcp.CallToolResult, any, error) {
		limit := 100
		if p.MaxItems > 0 {
			limit = p.MaxItems
		}
		var groups []any
		paginator := iam.NewListGroupsPaginator(client, &iam.ListGroupsInput{})
		for paginator.HasMorePages() && len(groups) < limit {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("iam:ListGroups", err)
			}
			for _, g := range page.Groups {
				groups = append(groups, g)
				if len(groups) >= limit {
					break
				}
			}
		}
		return awsutil.JSONResult(groups)
	})
}
