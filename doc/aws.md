# AWS Tools

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
| `aws_rds_events` | List recent RDS events (failovers, maintenance, reboots, storage issues) for an instance. Accepts a configurable time window in minutes (default: 1440 = 24 hours) |
| `aws_rds_pending_maintenance` | List pending maintenance actions across all RDS instances (engine upgrades, OS patches, security updates) |
| `aws_rds_snapshots` | List RDS snapshots (automated and manual) for a specific instance or all instances. Filterable by snapshot type |
| `aws_rds_read_replicas` | List RDS read replicas and their replication lag in seconds. Optionally filter by source instance |
| `aws_secrets_list` | List AWS Secrets Manager secrets. Optionally filter by name |
| `aws_secrets_get` | Get the value of a secret. If the secret is JSON, returns key-value pairs with optional key filtering |
| `aws_health_events` | List AWS Health events from the Personal Health Dashboard (end-of-support notices, deprecations, service incidents). Filterable by service and status. **Requires AWS Business or Enterprise Support plan** |
| `aws_docdb_ping` | Test the connection to a DocumentDB instance. Returns success status and round-trip latency in milliseconds |
| `aws_docdb_databases` | List databases on a DocumentDB instance with their size (MB) and empty status |
| `aws_docdb_collections` | List collections in a DocumentDB database with stats: document count, size (MB), average object size (bytes), index count and total index size (MB) |
| `aws_docdb_current_ops` | Show active operations on a DocumentDB instance (equivalent to `db.currentOp()`). Optionally filter by minimum running time with `min_secs` |
| `aws_docdb_server_status` | Show DocumentDB server telemetry (equivalent to `serverStatus` in MongoDB). Returns connections, operation counters (insert/query/update/delete/getmore/command), memory usage (resident/virtual MB), network I/O (bytes in/out, requests), global lock queue and active clients, uptime, host and version |

## IAM Permissions

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
