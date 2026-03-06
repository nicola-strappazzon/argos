# Percona Tools

Argos integrates with [Percona Toolkit](https://www.percona.com/software/database-tools/percona-toolkit) to run offline analysis against downloaded slow query logs and live MySQL instances. The `pt-query-digest`, `pt-index-usage`, and `pt-variable-advisor` binaries must be installed and available in `$PATH`.

## Tools

| Tool | Description |
|---|---|
| `pt_query_digest` | Run `pt-query-digest` on a downloaded slow query log and save the report to `/tmp/argos/pt-query-digest/` |
| `pt_index_usage` | Run `pt-index-usage` on a downloaded slow query log to find unused indexes. Saves the report to `/tmp/argos/pt-index-usage/`. Optionally filter by database |
| `pt_variable_advisor` | Run `pt-variable-advisor` against a MySQL/RDS instance and save the report to `/tmp/argos/pt-variable-advisor/`. The host and port can be obtained from `aws_rds_instances` |

## Typical Workflow

1. Use `aws_rds_logs` to list available slow query log files for an instance.
2. Use `aws_rds_log_download` to download the desired log file to `/tmp`.
3. Run `pt_query_digest` or `pt_index_usage` against the downloaded file.
4. Use `pt_variable_advisor` to get configuration recommendations directly from the live instance.
