// Package cloudwatch registers AWS CloudWatch read-only tools on an MCP server.
package cloudwatch

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/traffino/mcp-server/internal/aws/awsutil"
)

// Register adds CloudWatch tools to the MCP server.
func Register(s *mcp.Server, cfg aws.Config) {
	client := cloudwatch.NewFromConfig(cfg)

	// list_metrics
	type ListMetricsParams struct {
		Namespace  string `json:"namespace,omitempty"   jsonschema:"Optional CloudWatch namespace filter (e.g. AWS/EC2)"`
		MetricName string `json:"metric_name,omitempty" jsonschema:"Optional metric name filter"`
		MaxItems   int    `json:"max_items,omitempty"   jsonschema:"Maximum number of metrics to return (default 100)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_metrics",
		Description: "List CloudWatch metrics, optionally filtered by namespace or metric name.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListMetricsParams) (*mcp.CallToolResult, any, error) {
		limit := 100
		if p.MaxItems > 0 {
			limit = p.MaxItems
		}
		input := &cloudwatch.ListMetricsInput{}
		if p.Namespace != "" {
			input.Namespace = aws.String(p.Namespace)
		}
		if p.MetricName != "" {
			input.MetricName = aws.String(p.MetricName)
		}
		var metrics []any
		paginator := cloudwatch.NewListMetricsPaginator(client, input)
		for paginator.HasMorePages() && len(metrics) < limit {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("cloudwatch:ListMetrics", err)
			}
			for _, m := range page.Metrics {
				metrics = append(metrics, m)
				if len(metrics) >= limit {
					break
				}
			}
		}
		return awsutil.JSONResult(metrics)
	})

	// get_metric_statistics
	type GetMetricStatisticsParams struct {
		Namespace     string   `json:"namespace"       jsonschema:"The CloudWatch namespace (e.g. AWS/EC2)"`
		MetricName    string   `json:"metric_name"     jsonschema:"The metric name"`
		StartTime     string   `json:"start_time"      jsonschema:"Start time in RFC3339 format (e.g. 2024-01-01T00:00:00Z)"`
		EndTime       string   `json:"end_time"        jsonschema:"End time in RFC3339 format"`
		PeriodSeconds int      `json:"period_seconds"  jsonschema:"Granularity in seconds (must be multiple of 60 for standard metrics)"`
		Statistics    []string `json:"statistics"      jsonschema:"List of statistics: SampleCount Average Sum Minimum Maximum"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_metric_statistics",
		Description: "Get CloudWatch metric statistics for a given time range.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *GetMetricStatisticsParams) (*mcp.CallToolResult, any, error) {
		if p.Namespace == "" {
			return awsutil.ErrResult("namespace is required")
		}
		if p.MetricName == "" {
			return awsutil.ErrResult("metric_name is required")
		}
		if p.StartTime == "" {
			return awsutil.ErrResult("start_time is required")
		}
		if p.EndTime == "" {
			return awsutil.ErrResult("end_time is required")
		}
		if p.PeriodSeconds == 0 {
			return awsutil.ErrResult("period_seconds is required")
		}
		if len(p.Statistics) == 0 {
			return awsutil.ErrResult("statistics is required")
		}
		startTime, err := awsutil.ParseTime(p.StartTime)
		if err != nil {
			return awsutil.ErrResult("invalid start_time: " + err.Error())
		}
		endTime, err := awsutil.ParseTime(p.EndTime)
		if err != nil {
			return awsutil.ErrResult("invalid end_time: " + err.Error())
		}
		var stats []cwtypes.Statistic
		for _, st := range p.Statistics {
			stats = append(stats, cwtypes.Statistic(st))
		}
		out, err := client.GetMetricStatistics(ctx, &cloudwatch.GetMetricStatisticsInput{
			Namespace:  aws.String(p.Namespace),
			MetricName: aws.String(p.MetricName),
			StartTime:  &startTime,
			EndTime:    &endTime,
			Period:     aws.Int32(int32(p.PeriodSeconds)),
			Statistics: stats,
		})
		if err != nil {
			return awsutil.AWSErr("cloudwatch:GetMetricStatistics", err)
		}
		return awsutil.JSONResult(out.Datapoints)
	})

	// describe_alarms
	type DescribeAlarmsParams struct {
		MaxItems int `json:"max_items,omitempty" jsonschema:"Maximum number of alarms to return (default 100)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "describe_alarms",
		Description: "List CloudWatch alarms.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *DescribeAlarmsParams) (*mcp.CallToolResult, any, error) {
		limit := 100
		if p.MaxItems > 0 {
			limit = p.MaxItems
		}
		var alarms []any
		paginator := cloudwatch.NewDescribeAlarmsPaginator(client, &cloudwatch.DescribeAlarmsInput{})
		for paginator.HasMorePages() && len(alarms) < limit {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("cloudwatch:DescribeAlarms", err)
			}
			for _, a := range page.MetricAlarms {
				alarms = append(alarms, a)
				if len(alarms) >= limit {
					break
				}
			}
		}
		return awsutil.JSONResult(alarms)
	})
}
