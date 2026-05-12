// Package s3 registers AWS S3 read-only tools on an MCP server.
package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/traffino/mcp-server/internal/aws/awsutil"
)

// Register adds S3 tools to the MCP server.
func Register(srv *mcp.Server, cfg aws.Config) {
	client := s3.NewFromConfig(cfg)
	registerBucketSummary(srv, cfg, client)

	// list_buckets
	type ListBucketsParams struct {
		MaxItems int `json:"max_items,omitempty" jsonschema:"Maximum number of buckets to return (default 100)"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_buckets",
		Description: "List all S3 buckets in the account.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListBucketsParams) (*mcp.CallToolResult, any, error) {
		limit := 100
		if p.MaxItems > 0 {
			limit = p.MaxItems
		}
		var buckets []any
		paginator := s3.NewListBucketsPaginator(client, &s3.ListBucketsInput{})
		for paginator.HasMorePages() && len(buckets) < limit {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("s3:ListBuckets", err)
			}
			for _, b := range page.Buckets {
				buckets = append(buckets, b)
				if len(buckets) >= limit {
					break
				}
			}
		}
		return awsutil.JSONResult(buckets)
	})

	// get_bucket_location
	type GetBucketLocationParams struct {
		Bucket string `json:"bucket" jsonschema:"The name of the S3 bucket"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_bucket_location",
		Description: "Get the AWS Region where a bucket is located.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *GetBucketLocationParams) (*mcp.CallToolResult, any, error) {
		if p.Bucket == "" {
			return awsutil.ErrResult("bucket is required")
		}
		out, err := client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
			Bucket: aws.String(p.Bucket),
		})
		if err != nil {
			return awsutil.AWSErr("s3:GetBucketLocation", err)
		}
		return awsutil.JSONResult(map[string]any{
			"LocationConstraint": string(out.LocationConstraint),
		})
	})

	// list_objects
	type ListObjectsParams struct {
		Bucket   string `json:"bucket"              jsonschema:"The name of the S3 bucket"`
		Prefix   string `json:"prefix,omitempty"    jsonschema:"Optional prefix to filter objects"`
		MaxItems int    `json:"max_items,omitempty" jsonschema:"Maximum number of objects to return (default 100)"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_objects",
		Description: "List objects in an S3 bucket, optionally filtered by prefix.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListObjectsParams) (*mcp.CallToolResult, any, error) {
		if p.Bucket == "" {
			return awsutil.ErrResult("bucket is required")
		}
		limit := 100
		if p.MaxItems > 0 {
			limit = p.MaxItems
		}
		input := &s3.ListObjectsV2Input{
			Bucket: aws.String(p.Bucket),
		}
		if p.Prefix != "" {
			input.Prefix = aws.String(p.Prefix)
		}
		var objects []any
		paginator := s3.NewListObjectsV2Paginator(client, input)
		for paginator.HasMorePages() && len(objects) < limit {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("s3:ListObjectsV2", err)
			}
			for _, o := range page.Contents {
				objects = append(objects, o)
				if len(objects) >= limit {
					break
				}
			}
		}
		return awsutil.JSONResult(objects)
	})

	// head_object
	type HeadObjectParams struct {
		Bucket string `json:"bucket" jsonschema:"The name of the S3 bucket"`
		Key    string `json:"key"    jsonschema:"The object key"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "head_object",
		Description: "Get metadata for an S3 object (HEAD request, no body).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *HeadObjectParams) (*mcp.CallToolResult, any, error) {
		if p.Bucket == "" {
			return awsutil.ErrResult("bucket is required")
		}
		if p.Key == "" {
			return awsutil.ErrResult("key is required")
		}
		out, err := client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(p.Bucket),
			Key:    aws.String(p.Key),
		})
		if err != nil {
			return awsutil.AWSErr("s3:HeadObject", err)
		}
		return awsutil.JSONResult(out)
	})
}
