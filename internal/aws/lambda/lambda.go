// Package lambda registers AWS Lambda read-only tools on an MCP server.
package lambda

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/traffino/mcp-server/internal/aws/awsutil"
)

// Register adds Lambda tools to the MCP server.
func Register(s *mcp.Server, cfg aws.Config) {
	client := lambda.NewFromConfig(cfg)

	// list_functions
	type ListFunctionsParams struct {
		MaxItems int `json:"max_items,omitempty" jsonschema:"Maximum number of functions to return (default 100)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_functions",
		Description: "List Lambda functions in the account.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListFunctionsParams) (*mcp.CallToolResult, any, error) {
		limit := 100
		if p.MaxItems > 0 {
			limit = p.MaxItems
		}
		var functions []any
		paginator := lambda.NewListFunctionsPaginator(client, &lambda.ListFunctionsInput{})
		for paginator.HasMorePages() && len(functions) < limit {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("lambda:ListFunctions", err)
			}
			for _, f := range page.Functions {
				functions = append(functions, f)
				if len(functions) >= limit {
					break
				}
			}
		}
		return awsutil.JSONResult(functions)
	})

	// get_function
	type GetFunctionParams struct {
		FunctionName string `json:"function_name" jsonschema:"The Lambda function name or ARN"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_function",
		Description: "Get details for a specific Lambda function.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *GetFunctionParams) (*mcp.CallToolResult, any, error) {
		if p.FunctionName == "" {
			return awsutil.ErrResult("function_name is required")
		}
		out, err := client.GetFunction(ctx, &lambda.GetFunctionInput{
			FunctionName: aws.String(p.FunctionName),
		})
		if err != nil {
			return awsutil.AWSErr("lambda:GetFunction", err)
		}
		return awsutil.JSONResult(out)
	})

	// list_event_source_mappings
	type ListEventSourceMappingsParams struct {
		FunctionName string `json:"function_name,omitempty" jsonschema:"Optional Lambda function name or ARN to filter"`
		MaxItems     int    `json:"max_items,omitempty"     jsonschema:"Maximum number of mappings to return (default 100)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_event_source_mappings",
		Description: "List Lambda event source mappings, optionally filtered by function name.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListEventSourceMappingsParams) (*mcp.CallToolResult, any, error) {
		limit := 100
		if p.MaxItems > 0 {
			limit = p.MaxItems
		}
		input := &lambda.ListEventSourceMappingsInput{}
		if p.FunctionName != "" {
			input.FunctionName = aws.String(p.FunctionName)
		}
		var mappings []any
		paginator := lambda.NewListEventSourceMappingsPaginator(client, input)
		for paginator.HasMorePages() && len(mappings) < limit {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("lambda:ListEventSourceMappings", err)
			}
			for _, m := range page.EventSourceMappings {
				mappings = append(mappings, m)
				if len(mappings) >= limit {
					break
				}
			}
		}
		return awsutil.JSONResult(mappings)
	})
}
