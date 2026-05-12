// Package cloudwatch registers AWS CloudWatch read-only tools on an MCP server.
package cloudwatch

import (
	"context"
	"strconv"

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
	type Dimension struct {
		Name  string `json:"Name"  jsonschema:"Dimension name (e.g. BucketName, InstanceId)"`
		Value string `json:"Value" jsonschema:"Dimension value"`
	}
	type GetMetricStatisticsParams struct {
		Namespace     string      `json:"namespace"            jsonschema:"The CloudWatch namespace (e.g. AWS/EC2)"`
		MetricName    string      `json:"metric_name"          jsonschema:"The metric name"`
		StartTime     string      `json:"start_time"           jsonschema:"Start time in RFC3339 format (e.g. 2024-01-01T00:00:00Z)"`
		EndTime       string      `json:"end_time"             jsonschema:"End time in RFC3339 format"`
		PeriodSeconds int         `json:"period_seconds"       jsonschema:"Granularity in seconds (must be multiple of 60 for standard metrics)"`
		Statistics    []string    `json:"statistics"           jsonschema:"List of statistics: SampleCount Average Sum Minimum Maximum"`
		Dimensions    []Dimension `json:"dimensions,omitempty" jsonschema:"Optional list of dimensions to filter the metric (e.g. [{Name:BucketName,Value:my-bucket},{Name:StorageType,Value:StandardStorage}])"`
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
		var dims []cwtypes.Dimension
		for _, d := range p.Dimensions {
			if d.Name == "" || d.Value == "" {
				return awsutil.ErrResult("dimensions: Name and Value are required")
			}
			dims = append(dims, cwtypes.Dimension{
				Name:  aws.String(d.Name),
				Value: aws.String(d.Value),
			})
		}
		input := &cloudwatch.GetMetricStatisticsInput{
			Namespace:  aws.String(p.Namespace),
			MetricName: aws.String(p.MetricName),
			StartTime:  &startTime,
			EndTime:    &endTime,
			Period:     aws.Int32(int32(p.PeriodSeconds)),
			Statistics: stats,
		}
		if len(dims) > 0 {
			input.Dimensions = dims
		}
		out, err := client.GetMetricStatistics(ctx, input)
		if err != nil {
			return awsutil.AWSErr("cloudwatch:GetMetricStatistics", err)
		}
		return awsutil.JSONResult(out.Datapoints)
	})

	// get_metric_data
	type MetricDataQuery struct {
		ID            string      `json:"id"                   jsonschema:"Unique identifier for this query within the request (e.g. q1, m_bucket1). Must match ^[a-z][a-zA-Z0-9_]*$"`
		Namespace     string      `json:"namespace"            jsonschema:"The CloudWatch namespace (e.g. AWS/S3)"`
		MetricName    string      `json:"metric_name"          jsonschema:"The metric name (e.g. BucketSizeBytes)"`
		Dimensions    []Dimension `json:"dimensions,omitempty" jsonschema:"List of dimensions to filter the metric"`
		Statistic     string      `json:"statistic"            jsonschema:"Single statistic: SampleCount, Average, Sum, Minimum, Maximum"`
		PeriodSeconds int         `json:"period_seconds"       jsonschema:"Granularity in seconds (must be multiple of 60 for standard metrics)"`
		Label         string      `json:"label,omitempty"      jsonschema:"Optional human-readable label for the result series"`
	}
	type GetMetricDataParams struct {
		Queries   []MetricDataQuery `json:"queries"             jsonschema:"List of metric queries (up to 500 per call)"`
		StartTime string            `json:"start_time"          jsonschema:"Start time in RFC3339 format"`
		EndTime   string            `json:"end_time"            jsonschema:"End time in RFC3339 format"`
		ScanBy    string            `json:"scan_by,omitempty"   jsonschema:"Order datapoints are returned: TimestampDescending (default) or TimestampAscending"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_metric_data",
		Description: "Batch-fetch CloudWatch metric data (up to 500 queries per call) via GetMetricData.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *GetMetricDataParams) (*mcp.CallToolResult, any, error) {
		if len(p.Queries) == 0 {
			return awsutil.ErrResult("queries is required")
		}
		if len(p.Queries) > 500 {
			return awsutil.ErrResult("queries: max 500 per call")
		}
		if p.StartTime == "" {
			return awsutil.ErrResult("start_time is required")
		}
		if p.EndTime == "" {
			return awsutil.ErrResult("end_time is required")
		}
		startTime, err := awsutil.ParseTime(p.StartTime)
		if err != nil {
			return awsutil.ErrResult("invalid start_time: " + err.Error())
		}
		endTime, err := awsutil.ParseTime(p.EndTime)
		if err != nil {
			return awsutil.ErrResult("invalid end_time: " + err.Error())
		}
		queries := make([]cwtypes.MetricDataQuery, 0, len(p.Queries))
		for i, q := range p.Queries {
			if q.ID == "" {
				return awsutil.ErrResult("queries[" + strconv.Itoa(i) + "].id is required")
			}
			if q.Namespace == "" {
				return awsutil.ErrResult("queries[" + strconv.Itoa(i) + "].namespace is required")
			}
			if q.MetricName == "" {
				return awsutil.ErrResult("queries[" + strconv.Itoa(i) + "].metric_name is required")
			}
			if q.Statistic == "" {
				return awsutil.ErrResult("queries[" + strconv.Itoa(i) + "].statistic is required")
			}
			if q.PeriodSeconds == 0 {
				return awsutil.ErrResult("queries[" + strconv.Itoa(i) + "].period_seconds is required")
			}
			var dims []cwtypes.Dimension
			for _, d := range q.Dimensions {
				if d.Name == "" || d.Value == "" {
					return awsutil.ErrResult("queries[" + strconv.Itoa(i) + "].dimensions: Name and Value are required")
				}
				dims = append(dims, cwtypes.Dimension{
					Name:  aws.String(d.Name),
					Value: aws.String(d.Value),
				})
			}
			mdq := cwtypes.MetricDataQuery{
				Id: aws.String(q.ID),
				MetricStat: &cwtypes.MetricStat{
					Metric: &cwtypes.Metric{
						Namespace:  aws.String(q.Namespace),
						MetricName: aws.String(q.MetricName),
						Dimensions: dims,
					},
					Period: aws.Int32(int32(q.PeriodSeconds)),
					Stat:   aws.String(q.Statistic),
				},
			}
			if q.Label != "" {
				mdq.Label = aws.String(q.Label)
			}
			queries = append(queries, mdq)
		}
		input := &cloudwatch.GetMetricDataInput{
			MetricDataQueries: queries,
			StartTime:         &startTime,
			EndTime:           &endTime,
		}
		if p.ScanBy != "" {
			input.ScanBy = cwtypes.ScanBy(p.ScanBy)
		}
		var results []cwtypes.MetricDataResult
		paginator := cloudwatch.NewGetMetricDataPaginator(client, input)
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("cloudwatch:GetMetricData", err)
			}
			results = append(results, page.MetricDataResults...)
		}
		return awsutil.JSONResult(results)
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
