// Package eks registers AWS EKS read-only tools on an MCP server.
package eks

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/traffino/mcp-server/internal/aws/awsutil"
)

// Register adds EKS tools to the MCP server.
func Register(s *mcp.Server, cfg aws.Config) {
	client := eks.NewFromConfig(cfg)

	// list_eks_clusters
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_eks_clusters",
		Description: "List EKS cluster names.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ *struct{}) (*mcp.CallToolResult, any, error) {
		var names []string
		paginator := eks.NewListClustersPaginator(client, &eks.ListClustersInput{})
		for paginator.HasMorePages() && len(names) < 100 {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("eks:ListClusters", err)
			}
			for _, n := range page.Clusters {
				names = append(names, n)
				if len(names) >= 100 {
					break
				}
			}
		}
		return awsutil.JSONResult(names)
	})

	// describe_eks_cluster
	type DescribeEKSClusterParams struct {
		Name string `json:"name" jsonschema:"EKS cluster name"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "describe_eks_cluster",
		Description: "Describe an EKS cluster by name.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *DescribeEKSClusterParams) (*mcp.CallToolResult, any, error) {
		if p.Name == "" {
			return awsutil.ErrResult("name is required")
		}
		out, err := client.DescribeCluster(ctx, &eks.DescribeClusterInput{
			Name: aws.String(p.Name),
		})
		if err != nil {
			return awsutil.AWSErr("eks:DescribeCluster", err)
		}
		return awsutil.JSONResult(out.Cluster)
	})

	// list_nodegroups
	type ListNodegroupsParams struct {
		ClusterName string `json:"cluster_name" jsonschema:"EKS cluster name"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_nodegroups",
		Description: "List node group names for an EKS cluster.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListNodegroupsParams) (*mcp.CallToolResult, any, error) {
		if p.ClusterName == "" {
			return awsutil.ErrResult("cluster_name is required")
		}
		var names []string
		paginator := eks.NewListNodegroupsPaginator(client, &eks.ListNodegroupsInput{
			ClusterName: aws.String(p.ClusterName),
		})
		for paginator.HasMorePages() && len(names) < 100 {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("eks:ListNodegroups", err)
			}
			for _, n := range page.Nodegroups {
				names = append(names, n)
				if len(names) >= 100 {
					break
				}
			}
		}
		return awsutil.JSONResult(names)
	})
}
