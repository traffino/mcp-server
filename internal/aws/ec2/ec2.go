// Package ec2 registers AWS EC2 read-only tools on an MCP server.
package ec2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/traffino/mcp-server/internal/aws/awsutil"
)

// Register adds EC2 tools to the MCP server.
func Register(s *mcp.Server, cfg aws.Config) {
	client := ec2.NewFromConfig(cfg)

	// list_instances
	type ListInstancesParams struct {
		InstanceIDs []string `json:"instance_ids,omitempty" jsonschema:"Optional list of instance IDs to filter by"`
		MaxItems    int      `json:"max_items,omitempty"    jsonschema:"Maximum number of instances to return (default 100)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_instances",
		Description: "List EC2 instances, optionally filtered by instance IDs.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListInstancesParams) (*mcp.CallToolResult, any, error) {
		limit := 100
		if p.MaxItems > 0 {
			limit = p.MaxItems
		}
		input := &ec2.DescribeInstancesInput{}
		if len(p.InstanceIDs) > 0 {
			input.InstanceIds = p.InstanceIDs
		}
		var reservations []any
		paginator := ec2.NewDescribeInstancesPaginator(client, input)
		for paginator.HasMorePages() && len(reservations) < limit {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("ec2:DescribeInstances", err)
			}
			for _, r := range page.Reservations {
				reservations = append(reservations, r)
				if len(reservations) >= limit {
					break
				}
			}
		}
		return awsutil.JSONResult(reservations)
	})

	// describe_instance
	type DescribeInstanceParams struct {
		InstanceID string `json:"instance_id" jsonschema:"The EC2 instance ID"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "describe_instance",
		Description: "Describe a single EC2 instance by ID.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *DescribeInstanceParams) (*mcp.CallToolResult, any, error) {
		if p.InstanceID == "" {
			return awsutil.ErrResult("instance_id is required")
		}
		out, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: []string{p.InstanceID},
		})
		if err != nil {
			return awsutil.AWSErr("ec2:DescribeInstances", err)
		}
		return awsutil.JSONResult(out.Reservations)
	})

	// list_volumes
	type ListVolumesParams struct {
		MaxItems int `json:"max_items,omitempty" jsonschema:"Maximum number of volumes to return (default 100)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_volumes",
		Description: "List EBS volumes.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListVolumesParams) (*mcp.CallToolResult, any, error) {
		limit := 100
		if p.MaxItems > 0 {
			limit = p.MaxItems
		}
		var volumes []any
		paginator := ec2.NewDescribeVolumesPaginator(client, &ec2.DescribeVolumesInput{})
		for paginator.HasMorePages() && len(volumes) < limit {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("ec2:DescribeVolumes", err)
			}
			for _, v := range page.Volumes {
				volumes = append(volumes, v)
				if len(volumes) >= limit {
					break
				}
			}
		}
		return awsutil.JSONResult(volumes)
	})

	// describe_volume
	type DescribeVolumeParams struct {
		VolumeID string `json:"volume_id" jsonschema:"The EBS volume ID"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "describe_volume",
		Description: "Describe a single EBS volume by ID.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *DescribeVolumeParams) (*mcp.CallToolResult, any, error) {
		if p.VolumeID == "" {
			return awsutil.ErrResult("volume_id is required")
		}
		out, err := client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
			VolumeIds: []string{p.VolumeID},
		})
		if err != nil {
			return awsutil.AWSErr("ec2:DescribeVolumes", err)
		}
		return awsutil.JSONResult(out.Volumes)
	})

	// list_security_groups
	type ListSGParams struct {
		MaxItems int `json:"max_items,omitempty" jsonschema:"Maximum number of security groups to return (default 100)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_security_groups",
		Description: "List EC2 security groups.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListSGParams) (*mcp.CallToolResult, any, error) {
		limit := 100
		if p.MaxItems > 0 {
			limit = p.MaxItems
		}
		var groups []any
		paginator := ec2.NewDescribeSecurityGroupsPaginator(client, &ec2.DescribeSecurityGroupsInput{})
		for paginator.HasMorePages() && len(groups) < limit {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("ec2:DescribeSecurityGroups", err)
			}
			for _, g := range page.SecurityGroups {
				groups = append(groups, g)
				if len(groups) >= limit {
					break
				}
			}
		}
		return awsutil.JSONResult(groups)
	})

	// list_vpcs
	type ListVPCsParams struct {
		MaxItems int `json:"max_items,omitempty" jsonschema:"Maximum number of VPCs to return (default 100)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_vpcs",
		Description: "List VPCs in the account.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListVPCsParams) (*mcp.CallToolResult, any, error) {
		limit := 100
		if p.MaxItems > 0 {
			limit = p.MaxItems
		}
		var vpcs []any
		paginator := ec2.NewDescribeVpcsPaginator(client, &ec2.DescribeVpcsInput{})
		for paginator.HasMorePages() && len(vpcs) < limit {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("ec2:DescribeVpcs", err)
			}
			for _, v := range page.Vpcs {
				vpcs = append(vpcs, v)
				if len(vpcs) >= limit {
					break
				}
			}
		}
		return awsutil.JSONResult(vpcs)
	})

	// list_subnets
	type ListSubnetsParams struct {
		VPCID    string `json:"vpc_id,omitempty"    jsonschema:"Optional VPC ID to filter subnets"`
		MaxItems int    `json:"max_items,omitempty" jsonschema:"Maximum number of subnets to return (default 100)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_subnets",
		Description: "List subnets, optionally filtered by VPC ID.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListSubnetsParams) (*mcp.CallToolResult, any, error) {
		limit := 100
		if p.MaxItems > 0 {
			limit = p.MaxItems
		}
		input := &ec2.DescribeSubnetsInput{}
		if p.VPCID != "" {
			input.Filters = []ec2types.Filter{{
				Name:   aws.String("vpc-id"),
				Values: []string{p.VPCID},
			}}
		}
		var subnets []any
		paginator := ec2.NewDescribeSubnetsPaginator(client, input)
		for paginator.HasMorePages() && len(subnets) < limit {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("ec2:DescribeSubnets", err)
			}
			for _, sub := range page.Subnets {
				subnets = append(subnets, sub)
				if len(subnets) >= limit {
					break
				}
			}
		}
		return awsutil.JSONResult(subnets)
	})
}
