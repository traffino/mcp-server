// Package cloudwatchlogs registers AWS CloudWatch Logs read-only tools on an MCP server.
package cloudwatchlogs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/traffino/mcp-server/internal/aws/awsutil"
)

// Register adds CloudWatch Logs tools to the MCP server.
func Register(s *mcp.Server, cfg aws.Config) {
	client := cloudwatchlogs.NewFromConfig(cfg)

	// describe_log_groups
	type DescribeLogGroupsParams struct {
		NamePrefix string `json:"name_prefix,omitempty" jsonschema:"Optional log group name prefix filter"`
		MaxItems   int    `json:"max_items,omitempty"   jsonschema:"Maximum number of log groups to return (default 100)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "describe_log_groups",
		Description: "List CloudWatch Logs log groups, optionally filtered by name prefix.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *DescribeLogGroupsParams) (*mcp.CallToolResult, any, error) {
		limit := 100
		if p.MaxItems > 0 {
			limit = p.MaxItems
		}
		input := &cloudwatchlogs.DescribeLogGroupsInput{}
		if p.NamePrefix != "" {
			input.LogGroupNamePrefix = aws.String(p.NamePrefix)
		}
		var groups []any
		paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(client, input)
		for paginator.HasMorePages() && len(groups) < limit {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("cloudwatchlogs:DescribeLogGroups", err)
			}
			for _, g := range page.LogGroups {
				groups = append(groups, g)
				if len(groups) >= limit {
					break
				}
			}
		}
		return awsutil.JSONResult(groups)
	})

	// describe_log_streams
	type DescribeLogStreamsParams struct {
		LogGroupName string `json:"log_group_name"        jsonschema:"The log group name"`
		NamePrefix   string `json:"name_prefix,omitempty" jsonschema:"Optional log stream name prefix filter"`
		MaxItems     int    `json:"max_items,omitempty"   jsonschema:"Maximum number of log streams to return (default 100)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "describe_log_streams",
		Description: "List log streams within a CloudWatch Logs log group.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *DescribeLogStreamsParams) (*mcp.CallToolResult, any, error) {
		if p.LogGroupName == "" {
			return awsutil.ErrResult("log_group_name is required")
		}
		limit := 100
		if p.MaxItems > 0 {
			limit = p.MaxItems
		}
		input := &cloudwatchlogs.DescribeLogStreamsInput{
			LogGroupName: aws.String(p.LogGroupName),
		}
		if p.NamePrefix != "" {
			input.LogStreamNamePrefix = aws.String(p.NamePrefix)
		}
		var streams []any
		paginator := cloudwatchlogs.NewDescribeLogStreamsPaginator(client, input)
		for paginator.HasMorePages() && len(streams) < limit {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("cloudwatchlogs:DescribeLogStreams", err)
			}
			for _, st := range page.LogStreams {
				streams = append(streams, st)
				if len(streams) >= limit {
					break
				}
			}
		}
		return awsutil.JSONResult(streams)
	})

	// get_log_events
	type GetLogEventsParams struct {
		LogGroupName  string `json:"log_group_name"          jsonschema:"The log group name"`
		LogStreamName string `json:"log_stream_name"         jsonschema:"The log stream name"`
		StartTime     string `json:"start_time,omitempty"    jsonschema:"Optional start time in RFC3339 format"`
		EndTime       string `json:"end_time,omitempty"      jsonschema:"Optional end time in RFC3339 format"`
		Limit         int    `json:"limit,omitempty"         jsonschema:"Maximum number of log events to return"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_log_events",
		Description: "Get log events from a CloudWatch Logs log stream.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *GetLogEventsParams) (*mcp.CallToolResult, any, error) {
		if p.LogGroupName == "" {
			return awsutil.ErrResult("log_group_name is required")
		}
		if p.LogStreamName == "" {
			return awsutil.ErrResult("log_stream_name is required")
		}
		input := &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  aws.String(p.LogGroupName),
			LogStreamName: aws.String(p.LogStreamName),
		}
		if p.StartTime != "" {
			t, err := awsutil.ParseTime(p.StartTime)
			if err != nil {
				return awsutil.ErrResult("invalid start_time: " + err.Error())
			}
			ms := t.UnixMilli()
			input.StartTime = &ms
		}
		if p.EndTime != "" {
			t, err := awsutil.ParseTime(p.EndTime)
			if err != nil {
				return awsutil.ErrResult("invalid end_time: " + err.Error())
			}
			ms := t.UnixMilli()
			input.EndTime = &ms
		}
		if p.Limit > 0 {
			lim := int32(p.Limit)
			input.Limit = &lim
		}
		out, err := client.GetLogEvents(ctx, input)
		if err != nil {
			return awsutil.AWSErr("cloudwatchlogs:GetLogEvents", err)
		}
		return awsutil.JSONResult(out.Events)
	})
}
