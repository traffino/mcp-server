# AWS API Coverage

- **API**: AWS SDK for Go v2 (Service-spezifische APIs)
- **SDK Version**: aws-sdk-go-v2 (latest stable)
- **Letzter Check**: 2026-05-09
- **Scope**: readonly
- **Auth**: Static IAM Access Keys (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, optional `AWS_SESSION_TOKEN`), `AWS_REGION` Pflicht. Kein SSO/Profile/IMDS-Fallback.

## Tools

| Service | SDK-Operation | Tool-Name |
|---------|---------------|-----------|
| STS | GetCallerIdentity | get_caller_identity |
| EC2 | DescribeInstances | list_instances |
| EC2 | DescribeInstances (single) | describe_instance |
| EC2 | DescribeVolumes | list_volumes |
| EC2 | DescribeVolumes (single) | describe_volume |
| EC2 | DescribeSecurityGroups | list_security_groups |
| EC2 | DescribeVpcs | list_vpcs |
| EC2 | DescribeSubnets | list_subnets |
| S3 | ListBuckets | list_buckets |
| S3 | GetBucketLocation | get_bucket_location |
| S3 | ListObjectsV2 | list_objects |
| S3 | HeadObject | head_object |
| IAM | ListUsers | list_users |
| IAM | GetUser | get_user |
| IAM | ListRoles | list_roles |
| IAM | GetRole | get_role |
| IAM | ListPolicies | list_policies |
| IAM | ListGroups | list_groups |
| RDS | DescribeDBInstances | list_db_instances |
| RDS | DescribeDBInstances (single) | describe_db_instance |
| RDS | DescribeDBClusters | list_db_clusters |
| Lambda | ListFunctions | list_functions |
| Lambda | GetFunction | get_function |
| Lambda | ListEventSourceMappings | list_event_source_mappings |
| Route53 | ListHostedZones | list_hosted_zones |
| Route53 | ListResourceRecordSets | list_resource_record_sets |
| CloudWatch | ListMetrics | list_metrics |
| CloudWatch | GetMetricStatistics | get_metric_statistics |
| CloudWatch | DescribeAlarms | describe_alarms |
| CloudWatch Logs | DescribeLogGroups | describe_log_groups |
| CloudWatch Logs | DescribeLogStreams | describe_log_streams |
| CloudWatch Logs | GetLogEvents | get_log_events |
| ECS | ListClusters | list_ecs_clusters |
| ECS | DescribeClusters | describe_ecs_cluster |
| ECS | ListServices | list_services |
| ECS | DescribeServices | describe_service |
| ECS | ListTasks | list_tasks |
| EKS | ListClusters | list_eks_clusters |
| EKS | DescribeCluster | describe_eks_cluster |
| EKS | ListNodegroups | list_nodegroups |
| CloudFront | ListDistributions | list_distributions |
| CloudFront | GetDistribution | get_distribution |

**Total: 42 Tools**

## Out-of-Scope (V1)

- Alle schreibenden Operationen (Create*/Update*/Delete*/Put*/Modify*/Run*).
- Services jenseits des V1-Cuts: DynamoDB, SQS, SNS, KMS, Secrets Manager, Step Functions, AppSync, API Gateway, Cognito, etc. — bei Bedarf in V2 ergaenzen.
- AWS-Profile, SSO-Login, Assume-Role, IMDS — bewusst nicht unterstuetzt.

## Hinweise

- Pagination: viele List-Tools haben `max_items`-Param (Default 100), via SDK-Paginator.
- Time-Felder: ISO 8601 / RFC3339-Strings (z.B. `2026-05-01T00:00:00Z`).
- ECS und EKS Tool-Namen tragen Service-Prefix (`list_ecs_clusters`/`list_eks_clusters`) wegen Kollision.
- Region-Switching: einmalig per `AWS_REGION` Env. Multi-Region in einem Server-Lauf nicht unterstuetzt — bei Bedarf weiteren Container mit anderer Region starten.
