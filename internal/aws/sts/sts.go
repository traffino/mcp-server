// Package sts registers AWS STS read-only tools on an MCP server.
package sts

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/traffino/mcp-server/internal/aws/awsutil"
)

// Register adds STS tools to the MCP server.
func Register(s *mcp.Server, cfg aws.Config) {
	client := sts.NewFromConfig(cfg)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_caller_identity",
		Description: "Return the IAM identity of the credentials in use (ARN, UserId, Account). Smoke-test for auth.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ *struct{}) (*mcp.CallToolResult, any, error) {
		out, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			return awsutil.AWSErr("sts:GetCallerIdentity", err)
		}
		return awsutil.JSONResult(map[string]any{
			"Account": aws.ToString(out.Account),
			"Arn":     aws.ToString(out.Arn),
			"UserId":  aws.ToString(out.UserId),
		})
	})
}
