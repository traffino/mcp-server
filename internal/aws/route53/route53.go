// Package route53 registers AWS Route53 read-only tools on an MCP server.
package route53

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/traffino/mcp-server/internal/aws/awsutil"
)

// Register adds Route53 tools to the MCP server.
func Register(s *mcp.Server, cfg aws.Config) {
	client := route53.NewFromConfig(cfg)

	// list_hosted_zones
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_hosted_zones",
		Description: "List all Route53 hosted zones.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ *struct{}) (*mcp.CallToolResult, any, error) {
		var zones []any
		paginator := route53.NewListHostedZonesPaginator(client, &route53.ListHostedZonesInput{})
		for paginator.HasMorePages() && len(zones) < 100 {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("route53:ListHostedZones", err)
			}
			for _, z := range page.HostedZones {
				zones = append(zones, z)
				if len(zones) >= 100 {
					break
				}
			}
		}
		return awsutil.JSONResult(zones)
	})

	// list_resource_record_sets
	type ListResourceRecordSetsParams struct {
		HostedZoneID string `json:"hosted_zone_id"      jsonschema:"The Route53 hosted zone ID"`
		MaxItems     int    `json:"max_items,omitempty" jsonschema:"Maximum number of record sets to return (default 100)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_resource_record_sets",
		Description: "List resource record sets for a Route53 hosted zone.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListResourceRecordSetsParams) (*mcp.CallToolResult, any, error) {
		if p.HostedZoneID == "" {
			return awsutil.ErrResult("hosted_zone_id is required")
		}
		limit := 100
		if p.MaxItems > 0 {
			limit = p.MaxItems
		}
		var records []any
		paginator := route53.NewListResourceRecordSetsPaginator(client, &route53.ListResourceRecordSetsInput{
			HostedZoneId: aws.String(p.HostedZoneID),
		})
		for paginator.HasMorePages() && len(records) < limit {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("route53:ListResourceRecordSets", err)
			}
			for _, r := range page.ResourceRecordSets {
				records = append(records, r)
				if len(records) >= limit {
					break
				}
			}
		}
		return awsutil.JSONResult(records)
	})
}
