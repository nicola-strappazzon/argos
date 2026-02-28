# My Personal Database Administrator Assistant Server

Argos is a personal [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server written in Go that exposes AWS RDS observability tools to Claude (or any MCP-compatible client).

## What it does

This MCP server gives Claude direct access to your AWS infrastructure through a set of tools:

| Tool | Description |
|---|---|
| `aws_rds_instances` | List all RDS instances with engine, class, status, endpoint and MultiAZ info |
| `aws_rds_metrics` | Fetch CloudWatch metrics for a given instance (CPU, memory, storage, IOPS, latency, network). Auto-detects engine namespace (`AWS/RDS` vs `AWS/DocDB`) |
| `aws_rds_logs` | List available log files for a given RDS instance (name, size, last written) |
| `aws_rds_log_download` | Download a specific RDS log file and save it to `/tmp/aws_rds_logs/<instance>/<log_file>` |
| `aws_rds_parameter_groups` | List all RDS DB parameter groups (name, family, description, ARN) |
| `pt_query_digest` | Run `pt-query-digest` on a downloaded slow query log and save the report to `/tmp/pt-query-digest/` |

## Requirements

- Go 1.25+
- AWS credentials configured (via `~/.aws/credentials`, environment variables, or IAM role)
- [Percona Toolkit](https://www.percona.com/software/database-tools/percona-toolkit) — `pt-query-digest` must be available in `$PATH`
- Claude Code CLI

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `AWS_REGION` | Yes | AWS region to connect to (e.g. `eu-west-1`) |

## Installation

```bash
# Clone the repository
git clone https://github.com/nicola-strappazzon/argos.git
cd argos

# Install dependencies
go mod download

# Build the binary
go build -o argos .
```

## Register with Claude Code

```bash
claude mcp add argos --scope user --transport stdio --env AWS_REGION=eu-west-1 /path/to/mcp/argos
```

Replace `/path/to/mcp/argos` with the absolute path to the compiled binary.

To verify it's registered:

```bash
claude mcp list
```

## AWS Permissions

The IAM user or role used must have the following permissions:

```json
{
  "Effect": "Allow",
  "Action": [
    "rds:DescribeDBInstances",
    "rds:DescribeDBLogFiles",
    "rds:DownloadDBLogFilePortion",
    "rds:DescribeDBParameterGroups",
    "cloudwatch:GetMetricData"
  ],
  "Resource": "*"
}
```
