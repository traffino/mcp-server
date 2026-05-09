// Package ecs registers AWS ECS read-only tools on an MCP server.
package ecs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/traffino/mcp-server/internal/aws/awsutil"
)

// Register adds ECS tools to the MCP server.
func Register(s *mcp.Server, cfg aws.Config) {
	client := ecs.NewFromConfig(cfg)

	// list_ecs_clusters
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_ecs_clusters",
		Description: "List ECS cluster ARNs.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ *struct{}) (*mcp.CallToolResult, any, error) {
		var arns []string
		paginator := ecs.NewListClustersPaginator(client, &ecs.ListClustersInput{})
		for paginator.HasMorePages() && len(arns) < 100 {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("ecs:ListClusters", err)
			}
			for _, arn := range page.ClusterArns {
				arns = append(arns, arn)
				if len(arns) >= 100 {
					break
				}
			}
		}
		return awsutil.JSONResult(arns)
	})

	// describe_ecs_cluster
	type DescribeECSClusterParams struct {
		Cluster string `json:"cluster" jsonschema:"ECS cluster name or ARN"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "describe_ecs_cluster",
		Description: "Describe an ECS cluster by name or ARN.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *DescribeECSClusterParams) (*mcp.CallToolResult, any, error) {
		if p.Cluster == "" {
			return awsutil.ErrResult("cluster is required")
		}
		out, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
			Clusters: []string{p.Cluster},
		})
		if err != nil {
			return awsutil.AWSErr("ecs:DescribeClusters", err)
		}
		return awsutil.JSONResult(out.Clusters)
	})

	// list_services
	type ListServicesParams struct {
		Cluster string `json:"cluster" jsonschema:"ECS cluster name or ARN"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_services",
		Description: "List ECS service ARNs in a cluster.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListServicesParams) (*mcp.CallToolResult, any, error) {
		if p.Cluster == "" {
			return awsutil.ErrResult("cluster is required")
		}
		var arns []string
		paginator := ecs.NewListServicesPaginator(client, &ecs.ListServicesInput{
			Cluster: aws.String(p.Cluster),
		})
		for paginator.HasMorePages() && len(arns) < 100 {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("ecs:ListServices", err)
			}
			for _, arn := range page.ServiceArns {
				arns = append(arns, arn)
				if len(arns) >= 100 {
					break
				}
			}
		}
		return awsutil.JSONResult(arns)
	})

	// describe_service
	type DescribeServiceParams struct {
		Cluster string `json:"cluster" jsonschema:"ECS cluster name or ARN"`
		Service string `json:"service" jsonschema:"ECS service name or ARN"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "describe_service",
		Description: "Describe an ECS service within a cluster.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *DescribeServiceParams) (*mcp.CallToolResult, any, error) {
		if p.Cluster == "" {
			return awsutil.ErrResult("cluster is required")
		}
		if p.Service == "" {
			return awsutil.ErrResult("service is required")
		}
		out, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
			Cluster:  aws.String(p.Cluster),
			Services: []string{p.Service},
		})
		if err != nil {
			return awsutil.AWSErr("ecs:DescribeServices", err)
		}
		return awsutil.JSONResult(out.Services)
	})

	// list_tasks
	type ListTasksParams struct {
		Cluster     string `json:"cluster"               jsonschema:"ECS cluster name or ARN"`
		ServiceName string `json:"service_name,omitempty" jsonschema:"Optional service name to filter tasks"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_tasks",
		Description: "List ECS task ARNs in a cluster, optionally filtered by service.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListTasksParams) (*mcp.CallToolResult, any, error) {
		if p.Cluster == "" {
			return awsutil.ErrResult("cluster is required")
		}
		input := &ecs.ListTasksInput{
			Cluster: aws.String(p.Cluster),
		}
		if p.ServiceName != "" {
			input.ServiceName = aws.String(p.ServiceName)
		}
		var arns []string
		paginator := ecs.NewListTasksPaginator(client, input)
		for paginator.HasMorePages() && len(arns) < 100 {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("ecs:ListTasks", err)
			}
			for _, arn := range page.TaskArns {
				arns = append(arns, arn)
				if len(arns) >= 100 {
					break
				}
			}
		}
		return awsutil.JSONResult(arns)
	})
}
