# Argos — Personal DBA Assistant

All-seeing Argos, is a personal [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server written in Go that gives Claude direct, read-only access to your AWS RDS infrastructure.

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
| `aws_secrets_list` | List AWS Secrets Manager secrets. Optionally filter by name |
| `aws_secrets_get` | Get the value of a secret. If the secret is JSON, returns key-value pairs with optional key filtering |
| `aws_rds_events` | List recent RDS events (failovers, maintenance, reboots, storage issues) for an instance. Accepts a configurable time window in minutes (default: 1440 = 24 hours) |
| `aws_health_events` | List AWS Health events from the Personal Health Dashboard (end-of-support notices, deprecations, service incidents). Filterable by service and status. **Requires AWS Business or Enterprise Support plan** |
| `aws_rds_pending_maintenance` | List pending maintenance actions across all RDS instances (engine upgrades, OS patches, security updates) |
| `aws_rds_snapshots` | List RDS snapshots (automated and manual) for a specific instance or all instances. Filterable by snapshot type |
| `aws_rds_read_replicas` | List RDS read replicas and their replication lag in seconds. Optionally filter by source instance |
| `mysql_databases` | List databases on a MySQL instance with their size (MB), character set, collation and table count |
| `mysql_tables` | List tables within a database with engine, size (data/index/free), charset, collation, row format, estimated rows, fragmentation percentage, auto_increment, comment and timestamps |
| `mysql_describe_table` | Describe the columns of a table: type, nullability, default, charset, collation, key type, extra and comment |
| `mysql_table_indexes` | List indexes of a table with type, uniqueness, visibility, cardinality, columns (with position and prefix length) and size in MB |
| `mysql_table_foreign_keys` | List outgoing FKs (this table references others) and incoming FKs (other tables reference this table) with ON UPDATE/DELETE rules |
| `mysql_global_variables` | Run `SHOW GLOBAL VARIABLES` on a MySQL instance. Optionally filter by variable name using a `LIKE` pattern (e.g. `innodb%`) |
| `mysql_overflow` | Check AUTO_INCREMENT overflow risk for all tables in a database. Returns current value, max value, percentage used, and remaining capacity per column, sorted by percentage used descending |
| `mysql_status` | Run `SHOW ENGINE INNODB STATUS` and return parsed structured output: semaphores, latest deadlock (queries and victim), transactions, file I/O, log, buffer pool and row operations |
| `pt_query_digest` | Run `pt-query-digest` on a downloaded slow query log and save the report to `/tmp/argos/pt-query-digest/` |
| `pt_index_usage` | Run `pt-index-usage` on a downloaded slow query log to find unused indexes. Saves the report to `/tmp/argos/pt-index-usage/`. Optionally filter by database |
| `pt_variable_advisor` | Run `pt-variable-advisor` against a MySQL/RDS instance and save the report to `/tmp/argos/pt-variable-advisor/`. The host and port can be obtained from `aws_rds_instances` |

## Requirements

- Go 1.25+
- AWS credentials configured via `~/.aws/credentials`, environment variables, or IAM role
- [Percona Toolkit](https://www.percona.com/software/database-tools/percona-toolkit) — `pt-query-digest`, `pt-index-usage` and `pt-variable-advisor` must be in `$PATH`
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

## MySQL Credentials

Tools that connect directly to MySQL (e.g. `pt_variable_advisor`) read credentials from `~/.my.cnf`. Each RDS instance must have its own section named after the instance identifier:

```ini
[com-prd-mysql-general-node01]
host=com-prd-mysql-general-node01.xxxxxxxxxxxx.eu-west-1.rds.amazonaws.com
user=your_user
password=your_password
port=3306
```

The section name must match exactly the `db_instance_identifier` used in the tool call.

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
    "rds:DescribeEvents",
    "rds:DescribeDBSnapshots",
    "rds:DescribePendingMaintenanceActions",
    "cloudwatch:GetMetricData",
    "pi:DescribeDimensionKeys",
    "pi:GetResourceMetrics",
    "secretsmanager:ListSecrets",
    "secretsmanager:GetSecretValue"
  ],
  "Resource": "*"
}
```

> **Note:** `aws_health_events` additionally requires `health:DescribeEvents` and `health:DescribeEventDetails`, but these are only available with AWS Business or Enterprise Support plan.
