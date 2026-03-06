# Argos — Personal DataBase Assistant

All-seeing Argos, is a personal [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server written in Go that gives Claude direct, read-only access to your AWS RDS infrastructure.

**It never writes, modifies, or deletes anything.** Every tool is strictly observability: it reads from AWS APIs and CloudWatch, and writes only to your local `/tmp` directory when downloading log files for analysis.

> [!WARNING]
> **Diagnostic output only — not a substitute for expertise.** Acting on this tool's output without a solid understanding of database internals can cause data loss or outages. Always validate changes in a non-production environment and review them with a qualified DBA before applying anything to production. Use at your own risk.

## Tools

| Tool | Description |
|---|---|
| `aws_ec2_list` | List EC2 instances with name, instance ID, private/public IP, availability zone, instance type, and state. Optionally filter by Name tag (case-insensitive substring match) |
| `aws_rds_instances` | List all RDS instances: engine, version, instance class, status, endpoint, availability zone, MultiAZ, and Performance Insights status |
| `aws_rds_metrics` | Fetch the last 15 minutes of CloudWatch metrics for any instance: CPU, active connections, freeable memory, free storage, read/write IOPS, read/write latency, and network throughput. Auto-detects namespace (`AWS/RDS` vs `AWS/DocDB`) |
| `aws_rds_logs` | List available log files for an instance: name, size, and last written timestamp |
| `aws_rds_log_download` | Download a log file to `/tmp/argos/aws_rds_logs/<instance>/<log_file>` for local analysis |
| `aws_rds_parameter_groups` | List all user-customized parameters of the parameter group associated with a given RDS instance |
| `aws_rds_performance_insights` | Get the top 10 SQL queries and top 10 wait events by DB load average from Performance Insights for MySQL and PostgreSQL RDS instances. Accepts a configurable time window in minutes (default: 60) |
| `aws_secrets_list` | List AWS Secrets Manager secrets. Optionally filter by name |
| `aws_secrets_get` | Get the value of a secret. If the secret is JSON, returns key-value pairs with optional key filtering |
| `aws_rds_events` | List recent RDS events (failovers, maintenance, reboots, storage issues) for an instance. Accepts a configurable time window in minutes (default: 1440 = 24 hours) |
| `aws_health_events` | List AWS Health events from the Personal Health Dashboard (end-of-support notices, deprecations, service incidents). Filterable by service and status. **Requires AWS Business or Enterprise Support plan** |
| `aws_rds_pending_maintenance` | List pending maintenance actions across all RDS instances (engine upgrades, OS patches, security updates) |
| `aws_rds_snapshots` | List RDS snapshots (automated and manual) for a specific instance or all instances. Filterable by snapshot type |
| `aws_rds_read_replicas` | List RDS read replicas and their replication lag in seconds. Optionally filter by source instance |
| `aws_docdb_ping` | Test the connection to a DocumentDB instance. Returns success status and round-trip latency in milliseconds. Credentials are read from `~/.docdb` using the instance identifier as the section name |
| `aws_docdb_databases` | List databases on a DocumentDB instance with their size (MB) and empty status |
| `aws_docdb_collections` | List collections in a DocumentDB database with stats: document count, size (MB), average object size (bytes), index count and total index size (MB) |
| `aws_docdb_current_ops` | Show active operations on a DocumentDB instance (equivalent to `db.currentOp()`). Optionally filter by minimum running time with `min_secs` |
| `aws_docdb_server_status` | Show DocumentDB server telemetry (equivalent to `serverStatus` in MongoDB). Returns connections, operation counters (insert/query/update/delete/getmore/command), memory usage (resident/virtual MB), network I/O (bytes in/out, requests), global lock queue and active clients, uptime, host and version |
| `mysql_databases` | List databases on a MySQL instance with their size (MB), character set, collation and table count |
| `mysql_tables` | List tables within a database with engine, size (data/index/free), charset, collation, row format, estimated rows, fragmentation percentage, auto_increment, comment and timestamps |
| `mysql_describe_table` | Describe the columns of a table: type, nullability, default, charset, collation, key type, extra and comment |
| `mysql_table_indexes` | List indexes of a table with type, uniqueness, visibility, cardinality, columns (with position and prefix length) and size in MB |
| `mysql_table_foreign_keys` | List outgoing FKs (this table references others) and incoming FKs (other tables reference this table) with ON UPDATE/DELETE rules |
| `mysql_performance` | Analyze MySQL configuration variables and return tuning recommendations with status (`ok` / `warning`). Checks whether key variables are optimally set for the instance's resources, reporting current values and suggested changes. Checks: InnoDB buffer pool chunk size alignment (`innodb_buffer_pool_size` must be a multiple of `innodb_buffer_pool_instances × innodb_buffer_pool_chunk_size`) and chunk size ratio (optimal: 2–5% of buffer pool) |
| `mysql_health_check` | Run health checks on a MySQL instance and return key metrics with status (`ok` / `warning` / `critical`). Checks: InnoDB buffer pool hit rate, buffer pool size vs available RAM, thread cache hit rate, thread cache ratio (ok >= 10%), temporary tables on disk, InnoDB history list length, max connections usage, InnoDB dirty pages ratio (ok < 75%), open files utilization (ok < 85%), flushing logs ratio (ok < 5%), sort merge passes ratio (ok < 10%), and InnoDB redo log fill time (optimal: 45–75 min) |
| `mysql_ping` | Test the connection to a MySQL instance. Returns success status and round-trip latency in milliseconds |
| `mysql_processlist` | Run `SHOW FULL PROCESSLIST` on a MySQL instance. Idle connections (`Command=Sleep`) are excluded by default. Pass `include_idle: true` to show all |
| `mysql_explain` | Run `EXPLAIN` on a query and return the execution plan as structured rows. Optionally run `EXPLAIN ANALYZE` to include actual execution metrics (warning: executes the query) |
| `mysql_variables` | Run `SHOW GLOBAL VARIABLES` on a MySQL instance. Optionally filter by variable name using a `LIKE` pattern (e.g. `innodb%`) |
| `mysql_overflow` | Check AUTO_INCREMENT overflow risk for all tables in a database. Returns current value, max value, percentage used, and remaining capacity per column, sorted by percentage used descending |
| `mysql_innodb` | Run `SHOW ENGINE INNODB STATUS` and return parsed structured output: semaphores, latest deadlock (queries and victim), transactions, file I/O, log, buffer pool and row operations |
| `mysql_status` | Run `SHOW GLOBAL STATUS` on a MySQL instance. Optionally filter by variable name using a `LIKE` pattern (e.g. `Innodb%`, `Threads%`) |
| `postgresql_databases` | List databases on a PostgreSQL instance with their size (MB), encoding, collation, owner and connection limit |
| `postgresql_ping` | Test the connection to a PostgreSQL instance. Returns success status and round-trip latency in milliseconds. Credentials are read from `~/.pgpass` matching the instance identifier against the hostname |
| `postgresql_tables` | List tables within a PostgreSQL database with detailed info: schema, owner, access method, estimated row count, dead tuples, size (data/index/total), comment and last vacuum/analyze timestamps |
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

Tools that connect directly to MySQL read credentials from `~/.my.cnf`. Each RDS instance must have its own section named after the instance identifier:

```ini
[com-prd-mysql-general-node01]
host=com-prd-mysql-general-node01.xxxxxxxxxxxx.eu-west-1.rds.amazonaws.com
user=your_user
password=your_password
port=3306
```

The section name must match exactly the `db_instance_identifier` used in the tool call.

### Connecting via SSH tunnel

If the MySQL instance is not directly reachable, you can route the connection through an SSH bastion host by adding `ssh_*` fields to the same section:

```ini
[com-prd-mysql-general-node01]
host=com-prd-mysql-general-node01.xxxxxxxxxxxx.eu-west-1.rds.amazonaws.com
user=your_user
password=your_password
port=3306
ssh_host=bastion.example.com
ssh_user=ec2-user
ssh_key=~/.ssh/id_rsa
```

| Field | Required | Default | Description |
|---|---|---|---|
| `ssh_host` | Yes | — | Bastion host address |
| `ssh_user` | Yes | — | SSH username |
| `ssh_key` | Yes | — | Path to the private key file. Supports `~/` expansion |
| `ssh_port` | No | `22` | SSH port |

The tunnel is established in-process — no local port is opened and no external `ssh` process is required. If `ssh_host` is omitted, the connection is made directly as usual.

## DocumentDB Credentials

Tools that connect directly to DocumentDB read credentials from `~/.docdb`. Each instance must have its own section named after the instance identifier:

```ini
[my-docdb-instance-node01]
host=my-docdb-instance-node01.xxxxxxxxxxxx.eu-west-1.docdb.amazonaws.com
port=27017
user=docdbadmin
password=your_password
tls=true
tls_ca_file=/path/to/rds-combined-ca-bundle.pem
```

`tls` and `tls_ca_file` are optional. If `tls=true` and no `tls_ca_file` is provided, the system's default CA pool is used.

> **Note:** DocumentDB slow query profiling is not available via the `profile` command. To capture slow queries, enable `profiler=enabled` in the cluster parameter group and configure CloudWatch Logs export for the `profiler` log type.

## PostgreSQL Credentials

Tools that connect directly to PostgreSQL read credentials from `~/.pgpass`. Each RDS instance must have its own line using the standard format:

```
hostname:port:database:username:password
```

Example:

```
com-prd-psql-general-node01.xxxxxxxxxxxx.eu-west-1.rds.amazonaws.com:5432:postgres:your_user:your_password
```

The hostname is matched against the `db_instance_identifier` prefix, so `my-instance` will match `com-prd-psql-general-node01.xxxxxxxxxxxx.eu-west-1.rds.amazonaws.com`. The file must have permissions `600`:

```bash
chmod 600 ~/.pgpass
```

## AWS Permissions

Argos only requires read permissions. The IAM user or role must have:

```json
{
  "Effect": "Allow",
  "Action": [
    "ec2:DescribeInstances",
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
