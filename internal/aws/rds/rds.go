// Package rds registers AWS RDS read-only tools on an MCP server.
package rds

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/traffino/mcp-server/internal/aws/awsutil"
)

// Register adds RDS tools to the MCP server.
func Register(s *mcp.Server, cfg aws.Config) {
	client := rds.NewFromConfig(cfg)

	// list_db_instances
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_db_instances",
		Description: "List all RDS DB instances.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ *struct{}) (*mcp.CallToolResult, any, error) {
		var instances []any
		paginator := rds.NewDescribeDBInstancesPaginator(client, &rds.DescribeDBInstancesInput{})
		for paginator.HasMorePages() && len(instances) < 100 {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("rds:DescribeDBInstances", err)
			}
			for _, inst := range page.DBInstances {
				instances = append(instances, inst)
				if len(instances) >= 100 {
					break
				}
			}
		}
		return awsutil.JSONResult(instances)
	})

	// describe_db_instance
	type DescribeDBInstanceParams struct {
		DBInstanceIdentifier string `json:"db_instance_identifier" jsonschema:"The DB instance identifier"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "describe_db_instance",
		Description: "Describe a specific RDS DB instance by identifier.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *DescribeDBInstanceParams) (*mcp.CallToolResult, any, error) {
		if p.DBInstanceIdentifier == "" {
			return awsutil.ErrResult("db_instance_identifier is required")
		}
		out, err := client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(p.DBInstanceIdentifier),
		})
		if err != nil {
			return awsutil.AWSErr("rds:DescribeDBInstances", err)
		}
		return awsutil.JSONResult(out.DBInstances)
	})

	// list_db_clusters
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_db_clusters",
		Description: "List all RDS DB clusters (Aurora and multi-AZ clusters).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ *struct{}) (*mcp.CallToolResult, any, error) {
		var clusters []any
		paginator := rds.NewDescribeDBClustersPaginator(client, &rds.DescribeDBClustersInput{})
		for paginator.HasMorePages() && len(clusters) < 100 {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("rds:DescribeDBClusters", err)
			}
			for _, cl := range page.DBClusters {
				clusters = append(clusters, cl)
				if len(clusters) >= 100 {
					break
				}
			}
		}
		return awsutil.JSONResult(clusters)
	})
}
