---
date: 2026-05-09
type: concept
tags: [mcp, aws, server, architecture]
---

# aws-mcp-server (Phase C)

Eigenbau Go-MCP-Server fuer AWS-Account-Zugriff. Read-only, Static IAM Keys, Service-Cut V1.

## Abgrenzung zu Phase A

Phase A ([[refs/user/concepts/aws-mcp-server]]) registriert den gehosteten AWS Knowledge MCP — anonymes Doku-Wissen. Dieser Server ist Phase C: Zugriff auf den eigenen AWS-Account.

Phase A wird lokal ueber [[aws-docs-mcp-server]] (HTTP-Proxy, anonym) gewrapped, damit der Aggregator alle Backends einheitlich buendelt.

| | Phase A (Knowledge) | Phase C (dieser Server) |
|---|---|---|
| Wissen | "wie geht AWS Service X" | "was lebt in MEINEM Account" |
| Auth | keine | Static IAM Access Keys |
| Hosting | gehostet | self-hosted (Docker, lokal/baunach) |
| Modifikation | keine | read-only V1 |

## Architektur-Entscheidung: AWS SDK for Go v2

Das Repo-CLAUDE.md schreibt "keine externen Dependencies ausser MCP SDK und stdlib". Hetzner zeigt das Pattern: rohes `net/http` gegen die REST-API.

Fuer AWS bewusst gebrochen — nutzt `github.com/aws/aws-sdk-go-v2`. Begruendung:

- AWS SigV4 selbst zu implementieren ist ~200 Zeilen Crypto/Header-Code mit URI-Encoding-Edge-Cases. Fehlerquelle.
- AWS pflegt das SDK mit jeder API-Aenderung. Wir nicht.
- Service-Clients sind konsistent typisiert, schoen mit `aws.Config` injizierbar.

Konsequenz: das Repo hat damit (zusammen mit `personal/` das `modernc.org/sqlite` nutzt) jetzt zwei Sonderfaelle. CLAUDE.md (oben) wurde entsprechend angepasst.

## Service-Cut V1

| Service-Paket | Tools | AWS SDK Modul |
|---|---|---|
| `internal/aws/sts/` | get_caller_identity | `service/sts` |
| `internal/aws/ec2/` | list_instances, describe_instance, list_volumes, describe_volume, list_security_groups, list_vpcs, list_subnets | `service/ec2` |
| `internal/aws/s3/` | list_buckets, get_bucket_location, list_objects, head_object | `service/s3` |
| `internal/aws/iam/` | list_users, get_user, list_roles, get_role, list_policies, list_groups | `service/iam` |
| `internal/aws/rds/` | list_db_instances, describe_db_instance, list_db_clusters | `service/rds` |
| `internal/aws/lambda/` | list_functions, get_function, list_event_source_mappings | `service/lambda` |
| `internal/aws/route53/` | list_hosted_zones, list_resource_record_sets | `service/route53` |
| `internal/aws/cloudwatch/` | list_metrics, get_metric_statistics, describe_alarms | `service/cloudwatch` |
| `internal/aws/cloudwatchlogs/` | describe_log_groups, describe_log_streams, get_log_events | `service/cloudwatchlogs` |
| `internal/aws/ecs/` | list_clusters, describe_cluster, list_services, describe_service, list_tasks | `service/ecs` |
| `internal/aws/eks/` | list_clusters, describe_cluster, list_nodegroups | `service/eks` |
| `internal/aws/cloudfront/` | list_distributions, get_distribution | `service/cloudfront` |

EBS und VPC sind keine eigenen SDK-Pakete — Volumes/SGs/Subnets gehen via `service/ec2`.

## Auth + Konfiguration

| Env-Var | Pflicht | Default |
|---|---|---|
| `AWS_ACCESS_KEY_ID` | ja | — |
| `AWS_SECRET_ACCESS_KEY` | ja | — |
| `AWS_REGION` | ja | — |
| `AWS_SESSION_TOKEN` | nein | leer |
| `PORT` | nein | `:8000` |

`config.LoadDefaultConfig` wird NICHT genutzt (keine Profile/SSO/IMDS-Fallbacks erlaubt — wir wollen explizite Static Keys).

## Tool-Naming-Konvention

`<verb>_<resource>` mit AWS-Verben (`list`, `describe`, `get`):

- `list_*` — Sammlung ohne Detail (z.B. `list_instances` ruft `DescribeInstances` ohne Filter)
- `describe_*` — Detail eines Resource (`describe_instance` mit `instance_id`)
- `get_*` — Read-Operations die im SDK so heissen (z.B. `get_caller_identity`)

Service-Prefix bei Mehrdeutigkeit: `ec2_list_volumes` ist NICHT gewaehlt — Volumes leben im EC2-Namespace, aber Tool-Namen bleiben kurz (`list_volumes`). Bei tatsaechlicher Kollision (z.B. ECS und EKS haben beide `list_clusters`) prefixen wir: `list_ecs_clusters` / `list_eks_clusters`.

## Realm-Rollout

- baunach-Aggregator: in dieser Iteration konfiguriert (Compose-Eintrag separat, NICHT Teil dieses Branch).
- personal-Aggregator: vorbereitet, Env-Vars leer — User fuellt nach Bedarf.

## Smoke

`get_caller_identity` (STS) ist Health-Check beim Start nicht — der Server startet ohne API-Call. Aber als erstes Tool bei Live-Nutzung anrufen, um Auth zu verifizieren.

## Build-Status (V1)

- `make aws` baut sauber, Binary ~14.9MB.
- 42 Tools registriert (1 STS + 41 Service-Tools).
- Smoke-Test mit Dummy-Creds: Server bootet, `/health` antwortet `{"status":"ok","server":"aws","version":"1.0.0"}`. Live-Calls schlagen erwartungsgemaess mit InvalidClientToken fehl.
- Docker-Image-Build: `make docker-aws` (multi-stage, alpine + ca-certificates).

