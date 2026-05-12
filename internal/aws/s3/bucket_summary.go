package s3

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/traffino/mcp-server/internal/aws/awsutil"
)

// bucketSizeStorageTypes lists the StorageType dimension values used for the
// BucketSizeBytes metric. AWS publishes one datapoint per (bucket, storage type)
// roughly every 24h.
var bucketSizeStorageTypes = []string{
	"StandardStorage",
	"StandardIAStorage",
	"IntelligentTieringStorage",
	"GlacierStorage",
	"DeepArchiveStorage",
}

// registerBucketSummary adds the bucket_summary convenience tool to the server.
// It is split out from Register() to keep s3.go focused on raw API wrappers.
func registerBucketSummary(srv *mcp.Server, cfg aws.Config, client *s3.Client) {
	type BucketSummaryParams struct {
		Buckets []string `json:"buckets,omitempty" jsonschema:"Optional list of bucket names. If omitted, all buckets in the account are summarized."`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "s3_bucket_summary",
		Description: "Size and object count for one or more S3 buckets via CloudWatch BucketSizeBytes/NumberOfObjects (per-region GetMetricData, one roundtrip per region).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *BucketSummaryParams) (*mcp.CallToolResult, any, error) {
		names := p.Buckets
		if len(names) == 0 {
			paginator := s3.NewListBucketsPaginator(client, &s3.ListBucketsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					return awsutil.AWSErr("s3:ListBuckets", err)
				}
				for _, b := range page.Buckets {
					if b.Name != nil {
						names = append(names, *b.Name)
					}
				}
			}
		}
		if len(names) == 0 {
			return awsutil.JSONResult([]any{})
		}

		bucketRegion := make(map[string]string, len(names))
		for _, name := range names {
			loc, err := client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
				Bucket: aws.String(name),
			})
			if err != nil {
				return awsutil.AWSErr("s3:GetBucketLocation("+name+")", err)
			}
			region := string(loc.LocationConstraint)
			if region == "" {
				region = "us-east-1"
			}
			bucketRegion[name] = region
		}

		byRegion := make(map[string][]string)
		for _, name := range names {
			r := bucketRegion[name]
			byRegion[r] = append(byRegion[r], name)
		}

		end := time.Now().UTC()
		start := end.Add(-72 * time.Hour)

		type bucketResult struct {
			Name             string           `json:"name"`
			Region           string           `json:"region"`
			SizeBytes        int64            `json:"size_bytes"`
			SizeHuman        string           `json:"size_human"`
			ObjectCount      int64            `json:"object_count"`
			StorageBreakdown map[string]int64 `json:"storage_breakdown"`
		}
		results := make(map[string]*bucketResult, len(names))
		for _, name := range names {
			results[name] = &bucketResult{
				Name:             name,
				Region:           bucketRegion[name],
				StorageBreakdown: make(map[string]int64, len(bucketSizeStorageTypes)),
			}
		}

		for region, buckets := range byRegion {
			cwClient := cloudwatch.NewFromConfig(cfg, func(o *cloudwatch.Options) {
				o.Region = region
			})

			type queryKey struct {
				bucket  string
				storage string // empty = object count
			}
			idToKey := make(map[string]queryKey)
			queries := make([]cwtypes.MetricDataQuery, 0, len(buckets)*(len(bucketSizeStorageTypes)+1))

			for i, name := range buckets {
				for j, st := range bucketSizeStorageTypes {
					id := fmt.Sprintf("s%d_%d", i, j)
					idToKey[id] = queryKey{bucket: name, storage: st}
					queries = append(queries, cwtypes.MetricDataQuery{
						Id: aws.String(id),
						MetricStat: &cwtypes.MetricStat{
							Metric: &cwtypes.Metric{
								Namespace:  aws.String("AWS/S3"),
								MetricName: aws.String("BucketSizeBytes"),
								Dimensions: []cwtypes.Dimension{
									{Name: aws.String("BucketName"), Value: aws.String(name)},
									{Name: aws.String("StorageType"), Value: aws.String(st)},
								},
							},
							Period: aws.Int32(86400),
							Stat:   aws.String("Average"),
						},
					})
				}
				id := fmt.Sprintf("n%d", i)
				idToKey[id] = queryKey{bucket: name, storage: ""}
				queries = append(queries, cwtypes.MetricDataQuery{
					Id: aws.String(id),
					MetricStat: &cwtypes.MetricStat{
						Metric: &cwtypes.Metric{
							Namespace:  aws.String("AWS/S3"),
							MetricName: aws.String("NumberOfObjects"),
							Dimensions: []cwtypes.Dimension{
								{Name: aws.String("BucketName"), Value: aws.String(name)},
								{Name: aws.String("StorageType"), Value: aws.String("AllStorageTypes")},
							},
						},
						Period: aws.Int32(86400),
						Stat:   aws.String("Average"),
					},
				})
			}

			for offset := 0; offset < len(queries); offset += 500 {
				upper := offset + 500
				if upper > len(queries) {
					upper = len(queries)
				}
				input := &cloudwatch.GetMetricDataInput{
					MetricDataQueries: queries[offset:upper],
					StartTime:         &start,
					EndTime:           &end,
					ScanBy:            cwtypes.ScanByTimestampDescending,
				}
				paginator := cloudwatch.NewGetMetricDataPaginator(cwClient, input)
				for paginator.HasMorePages() {
					page, err := paginator.NextPage(ctx)
					if err != nil {
						return awsutil.AWSErr("cloudwatch:GetMetricData("+region+")", err)
					}
					for _, r := range page.MetricDataResults {
						if r.Id == nil || len(r.Values) == 0 {
							continue
						}
						key, ok := idToKey[*r.Id]
						if !ok {
							continue
						}
						val := int64(r.Values[0])
						bucket := results[key.bucket]
						if key.storage == "" {
							bucket.ObjectCount = val
						} else if val > 0 {
							bucket.StorageBreakdown[key.storage] = val
						}
					}
				}
			}
		}

		out := make([]*bucketResult, 0, len(names))
		for _, name := range names {
			r := results[name]
			var total int64
			for _, v := range r.StorageBreakdown {
				total += v
			}
			r.SizeBytes = total
			r.SizeHuman = humanBytes(total)
			out = append(out, r)
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })

		return awsutil.JSONResult(out)
	})
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return strconv.FormatInt(n, 10) + " B"
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	suffix := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}[exp]
	return fmt.Sprintf("%.2f %s", float64(n)/float64(div), suffix)
}
