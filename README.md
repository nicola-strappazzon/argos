# Argos — Personal DBA Assistant

Argos is a personal [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server written in Go that gives Claude direct, read-only access to your AWS RDS infrastructure.

**It never writes, modifies, or deletes anything.** Every tool is strictly observability: it reads from AWS APIs and CloudWatch, and writes only to your local `/tmp` directory when downloading log files for analysis.

## Tools

| Tool | Description |
|---|---|
| `aws_rds_instances` | List all RDS instances: engine, version, instance class, status, endpoint, availability zone, MultiAZ, and Performance Insights status |
| `aws_rds_metrics` | Fetch the last 15 minutes of CloudWatch metrics for any instance: CPU, active connections, freeable memory, free storage, read/write IOPS, read/write latency, and network throughput. Auto-detects namespace (`AWS/RDS` vs `AWS/DocDB`) |
| `aws_rds_logs` | List available log files for an instance: name, size, and last written timestamp |
| `aws_rds_log_download` | Download a log file to `/tmp/argos/aws_rds_logs/<instance>/<log_file>` for local analysis |
| `aws_rds_parameter_groups` | List all user-customized parameters of the parameter group associated with a given RDS instance |
| `aws_rds_performance_insights` | Get the top 10 SQL queries and top 10 wait events by DB load average from Performance Insights. Accepts a configurable time window in minutes (default: 60). Supports RDS and DocumentDB |
| `pt_query_digest` | Run `pt-query-digest` on a downloaded slow query log and save the report to `/tmp/argos/pt-query-digest/` |

## Requirements

- Go 1.25+
- AWS credentials configured via `~/.aws/credentials`, environment variables, or IAM role
- [Percona Toolkit](https://www.percona.com/software/database-tools/percona-toolkit) — `pt-query-digest` must be in `$PATH`
- Claude Code CLI

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `AWS_REGION` | Yes | AWS region to connect to (e.g. `eu-west-1`) |

## Installation

```bash
git clone https://github.com/nicola-strappazzon/argos.git
cd argos
go mod download
go build -o argos .
```

## Register with Claude Code

```bash
claude mcp add argos \
  --scope user \
  --transport stdio \
  --env AWS_REGION=eu-west-1 \
  -- \
  /path/to/argos/argos
```

Replace `/path/to/argos/argos` with the absolute path to the compiled binary.

To verify it's registered:

```bash
claude mcp list
```

## AWS Permissions

Argos only requires read permissions. The IAM user or role must have:

```json
{
  "Effect": "Allow",
  "Action": [
    "rds:DescribeDBInstances",
    "rds:DescribeDBLogFiles",
    "rds:DownloadDBLogFilePortion",
    "rds:DescribeDBParameterGroups",
    "rds:DescribeDBParameters",
    "cloudwatch:GetMetricData",
    "pi:DescribeDimensionKeys",
    "pi:GetResourceMetrics"
  ],
  "Resource": "*"
}
```
