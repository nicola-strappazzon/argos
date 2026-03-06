# Argos — Personal DataBase Assistant

All-seeing Argos, is a personal [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server written in Go that gives Claude direct, read-only access to your AWS RDS infrastructure.

**It never writes, modifies, or deletes anything.** Every tool is strictly observability: it reads from AWS APIs and CloudWatch, and writes only to your local `/tmp` directory when downloading log files for analysis.

> [!WARNING]
> **Diagnostic output only — not a substitute for expertise.** Acting on this tool's output without a solid understanding of database internals can cause data loss or outages. Always validate changes in a non-production environment and review them with a qualified DBA before applying anything to production. Use at your own risk.

## Tools

| Group | Description | Docs |
|---|---|---|
| **AWS** | Inspect EC2 instances, RDS and DocumentDB clusters, CloudWatch metrics, slow query logs, parameter groups, Performance Insights, snapshots, read replicas, pending maintenance, health dashboard alerts, and Secrets Manager | [doc/aws.md](doc/aws.md) |
| **MySQL** | Connect directly to MySQL instances: explore databases, tables, indexes, and foreign keys; inspect processes, InnoDB internals, global variables and status; run health checks, performance tuning analysis, and schema validation | [doc/mysql.md](doc/mysql.md) |
| **PostgreSQL** | Connect directly to PostgreSQL instances: list databases and tables with size, bloat, and vacuum statistics | [doc/postgresql.md](doc/postgresql.md) |
| **Percona** | Run Percona Toolkit utilities against slow query logs and live instances: query digest, index usage analysis, and variable tuning advice | [doc/percona.md](doc/percona.md) |

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
