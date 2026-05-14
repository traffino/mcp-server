// Package bedrock registers AWS Bedrock (control plane) read-only tools on an MCP server.
package bedrock

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/traffino/mcp-server/internal/aws/awsutil"
)

const maxItems = 200

// Register adds Bedrock control-plane tools to the MCP server.
//
// Scope: Foundation Models only. Bedrock Agents / Knowledge Bases are not exposed.
func Register(s *mcp.Server, cfg aws.Config) {
	baseClient := bedrock.NewFromConfig(cfg)

	clientFor := func(region string) *bedrock.Client {
		if region == "" || region == cfg.Region {
			return baseClient
		}
		regional := cfg
		regional.Region = region
		return bedrock.NewFromConfig(regional)
	}

	// list_foundation_models
	type ListFMParams struct {
		Provider       string `json:"provider,omitempty" jsonschema:"Optional provider filter (e.g. 'anthropic', 'amazon', 'meta')."`
		OutputModality string `json:"output_modality,omitempty" jsonschema:"Optional output modality: TEXT, IMAGE, or EMBEDDING."`
		InferenceType  string `json:"inference_type,omitempty" jsonschema:"Optional inference type: ON_DEMAND or PROVISIONED."`
		Region         string `json:"region,omitempty" jsonschema:"Optional region override."`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_foundation_models",
		Description: "List Bedrock foundation models available in a region, with optional provider/modality/inference filters.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListFMParams) (*mcp.CallToolResult, any, error) {
		client := clientFor(p.Region)
		in := &bedrock.ListFoundationModelsInput{}
		if p.Provider != "" {
			in.ByProvider = aws.String(p.Provider)
		}
		if p.OutputModality != "" {
			in.ByOutputModality = bedrocktypes.ModelModality(p.OutputModality)
		}
		if p.InferenceType != "" {
			in.ByInferenceType = bedrocktypes.InferenceType(p.InferenceType)
		}
		out, err := client.ListFoundationModels(ctx, in)
		if err != nil {
			return awsutil.AWSErr("bedrock:ListFoundationModels", err)
		}
		return awsutil.JSONResult(out.ModelSummaries)
	})

	// get_foundation_model
	type GetFMParams struct {
		ModelIdentifier string `json:"model_identifier" jsonschema:"Model id or ARN (e.g. 'anthropic.claude-3-5-sonnet-20240620-v1:0'). Required."`
		Region          string `json:"region,omitempty" jsonschema:"Optional region override."`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_foundation_model",
		Description: "Get details for a specific Bedrock foundation model (modalities, customizations supported, inference types).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *GetFMParams) (*mcp.CallToolResult, any, error) {
		if p.ModelIdentifier == "" {
			return awsutil.ErrResult("model_identifier is required")
		}
		client := clientFor(p.Region)
		out, err := client.GetFoundationModel(ctx, &bedrock.GetFoundationModelInput{
			ModelIdentifier: aws.String(p.ModelIdentifier),
		})
		if err != nil {
			return awsutil.AWSErr("bedrock:GetFoundationModel", err)
		}
		return awsutil.JSONResult(out.ModelDetails)
	})

	// list_inference_profiles
	type ListProfilesParams struct {
		TypeEquals string `json:"type_equals,omitempty" jsonschema:"Optional filter: SYSTEM_DEFINED (cross-region) or APPLICATION (custom)."`
		Region     string `json:"region,omitempty" jsonschema:"Optional region override."`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_inference_profiles",
		Description: "List Bedrock inference profiles in a region (cross-region routing endpoints and application profiles).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListProfilesParams) (*mcp.CallToolResult, any, error) {
		client := clientFor(p.Region)
		in := &bedrock.ListInferenceProfilesInput{}
		if p.TypeEquals != "" {
			in.TypeEquals = bedrocktypes.InferenceProfileType(p.TypeEquals)
		}
		var profiles []any
		paginator := bedrock.NewListInferenceProfilesPaginator(client, in)
		for paginator.HasMorePages() && len(profiles) < maxItems {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("bedrock:ListInferenceProfiles", err)
			}
			for _, pr := range page.InferenceProfileSummaries {
				profiles = append(profiles, pr)
				if len(profiles) >= maxItems {
					break
				}
			}
		}
		return awsutil.JSONResult(profiles)
	})
}
