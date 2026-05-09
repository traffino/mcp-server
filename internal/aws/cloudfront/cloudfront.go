// Package cloudfront registers AWS CloudFront read-only tools on an MCP server.
package cloudfront

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/traffino/mcp-server/internal/aws/awsutil"
)

// Register adds CloudFront tools to the MCP server.
func Register(s *mcp.Server, cfg aws.Config) {
	client := cloudfront.NewFromConfig(cfg)

	// list_distributions
	type ListDistributionsParams struct {
		MaxItems int `json:"max_items,omitempty" jsonschema:"Maximum number of distributions to return (default 100)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_distributions",
		Description: "List CloudFront distributions.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListDistributionsParams) (*mcp.CallToolResult, any, error) {
		limit := 100
		if p.MaxItems > 0 {
			limit = p.MaxItems
		}
		var distros []any
		paginator := cloudfront.NewListDistributionsPaginator(client, &cloudfront.ListDistributionsInput{})
		for paginator.HasMorePages() && len(distros) < limit {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("cloudfront:ListDistributions", err)
			}
			if page.DistributionList != nil {
				for _, d := range page.DistributionList.Items {
					distros = append(distros, d)
					if len(distros) >= limit {
						break
					}
				}
			}
		}
		return awsutil.JSONResult(distros)
	})

	// get_distribution
	type GetDistributionParams struct {
		ID string `json:"id" jsonschema:"The CloudFront distribution ID"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_distribution",
		Description: "Get details for a specific CloudFront distribution.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *GetDistributionParams) (*mcp.CallToolResult, any, error) {
		if p.ID == "" {
			return awsutil.ErrResult("id is required")
		}
		out, err := client.GetDistribution(ctx, &cloudfront.GetDistributionInput{
			Id: aws.String(p.ID),
		})
		if err != nil {
			return awsutil.AWSErr("cloudfront:GetDistribution", err)
		}
		return awsutil.JSONResult(out.Distribution)
	})
}
