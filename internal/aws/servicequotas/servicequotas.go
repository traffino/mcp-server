// Package servicequotas registers AWS Service Quotas read-only tools on an MCP server.
package servicequotas

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/traffino/mcp-server/internal/aws/awsutil"
)

const maxItems = 200

// Register adds Service Quotas tools to the MCP server.
func Register(s *mcp.Server, cfg aws.Config) {
	baseClient := servicequotas.NewFromConfig(cfg)

	clientFor := func(region string) *servicequotas.Client {
		if region == "" || region == cfg.Region {
			return baseClient
		}
		regional := cfg
		regional.Region = region
		return servicequotas.NewFromConfig(regional)
	}

	// list_service_quotas
	type ListParams struct {
		ServiceCode string `json:"service_code" jsonschema:"AWS service code (e.g. 'bedrock', 'ec2', 'lambda'). Required."`
		Region      string `json:"region,omitempty" jsonschema:"Optional region override. Defaults to the server's AWS_REGION."`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_service_quotas",
		Description: "List applied (current) service quotas for an AWS service in a region. Use service_code='bedrock' for Bedrock model quotas (TPM/RPM per foundation model).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListParams) (*mcp.CallToolResult, any, error) {
		if p.ServiceCode == "" {
			return awsutil.ErrResult("service_code is required")
		}
		client := clientFor(p.Region)
		var quotas []any
		paginator := servicequotas.NewListServiceQuotasPaginator(client, &servicequotas.ListServiceQuotasInput{
			ServiceCode: aws.String(p.ServiceCode),
		})
		for paginator.HasMorePages() && len(quotas) < maxItems {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("servicequotas:ListServiceQuotas", err)
			}
			for _, q := range page.Quotas {
				quotas = append(quotas, q)
				if len(quotas) >= maxItems {
					break
				}
			}
		}
		return awsutil.JSONResult(quotas)
	})

	// list_aws_default_service_quotas
	type ListDefaultsParams struct {
		ServiceCode string `json:"service_code" jsonschema:"AWS service code. Required."`
		Region      string `json:"region,omitempty" jsonschema:"Optional region override."`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_aws_default_service_quotas",
		Description: "List AWS default (account-creation) service quotas. Compare against list_service_quotas to detect granted quota increases.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *ListDefaultsParams) (*mcp.CallToolResult, any, error) {
		if p.ServiceCode == "" {
			return awsutil.ErrResult("service_code is required")
		}
		client := clientFor(p.Region)
		var quotas []any
		paginator := servicequotas.NewListAWSDefaultServiceQuotasPaginator(client, &servicequotas.ListAWSDefaultServiceQuotasInput{
			ServiceCode: aws.String(p.ServiceCode),
		})
		for paginator.HasMorePages() && len(quotas) < maxItems {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return awsutil.AWSErr("servicequotas:ListAWSDefaultServiceQuotas", err)
			}
			for _, q := range page.Quotas {
				quotas = append(quotas, q)
				if len(quotas) >= maxItems {
					break
				}
			}
		}
		return awsutil.JSONResult(quotas)
	})

	// get_service_quota
	type GetParams struct {
		ServiceCode string `json:"service_code" jsonschema:"AWS service code. Required."`
		QuotaCode   string `json:"quota_code" jsonschema:"Quota code (e.g. 'L-12345678'). Required."`
		Region      string `json:"region,omitempty" jsonschema:"Optional region override."`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_service_quota",
		Description: "Get the applied value of a specific service quota by service_code + quota_code.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, p *GetParams) (*mcp.CallToolResult, any, error) {
		if p.ServiceCode == "" || p.QuotaCode == "" {
			return awsutil.ErrResult("service_code and quota_code are required")
		}
		client := clientFor(p.Region)
		out, err := client.GetServiceQuota(ctx, &servicequotas.GetServiceQuotaInput{
			ServiceCode: aws.String(p.ServiceCode),
			QuotaCode:   aws.String(p.QuotaCode),
		})
		if err != nil {
			return awsutil.AWSErr("servicequotas:GetServiceQuota", err)
		}
		return awsutil.JSONResult(out.Quota)
	})
}
