package main

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"

	"github.com/traffino/mcp-server/internal/aws/cloudfront"
	"github.com/traffino/mcp-server/internal/aws/cloudwatch"
	"github.com/traffino/mcp-server/internal/aws/cloudwatchlogs"
	"github.com/traffino/mcp-server/internal/aws/ec2"
	"github.com/traffino/mcp-server/internal/aws/ecs"
	"github.com/traffino/mcp-server/internal/aws/eks"
	"github.com/traffino/mcp-server/internal/aws/iam"
	"github.com/traffino/mcp-server/internal/aws/lambda"
	"github.com/traffino/mcp-server/internal/aws/rds"
	"github.com/traffino/mcp-server/internal/aws/route53"
	"github.com/traffino/mcp-server/internal/aws/s3"
	"github.com/traffino/mcp-server/internal/aws/sts"
	"github.com/traffino/mcp-server/internal/config"
	"github.com/traffino/mcp-server/internal/server"
)

func main() {
	accessKey := config.Require("AWS_ACCESS_KEY_ID")
	secretKey := config.Require("AWS_SECRET_ACCESS_KEY")
	region := config.Require("AWS_REGION")
	sessionToken := config.Get("AWS_SESSION_TOKEN", "")

	cfg := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, sessionToken),
	}

	srv := server.New("aws", "1.0.0")
	s := srv.MCPServer()

	sts.Register(s, cfg)
	ec2.Register(s, cfg)
	s3.Register(s, cfg)
	iam.Register(s, cfg)
	rds.Register(s, cfg)
	lambda.Register(s, cfg)
	route53.Register(s, cfg)
	cloudwatch.Register(s, cfg)
	cloudwatchlogs.Register(s, cfg)
	ecs.Register(s, cfg)
	eks.Register(s, cfg)
	cloudfront.Register(s, cfg)

	srv.ListenAndServe(config.Get("PORT", ":8000"))
}
