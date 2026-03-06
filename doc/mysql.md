# MySQL Tools

## Tools

| Tool | Description |
|---|---|
| `mysql_ping` | Test the connection to a MySQL instance. Returns success status and round-trip latency in milliseconds |
| `mysql_databases` | List databases on a MySQL instance with their size (MB), character set, collation and table count |
| `mysql_tables` | List tables within a database with engine, size (data/index/free), charset, collation, row format, estimated rows, fragmentation percentage, auto_increment, comment and timestamps |
| `mysql_describe_table` | Describe the columns of a table: type, nullability, default, charset, collation, key type, extra and comment |
| `mysql_table_indexes` | List indexes of a table with type, uniqueness, visibility, cardinality, columns (with position and prefix length) and size in MB |
| `mysql_table_foreign_keys` | List outgoing FKs (this table references others) and incoming FKs (other tables reference this table) with ON UPDATE/DELETE rules |
| `mysql_processlist` | Run `SHOW FULL PROCESSLIST` on a MySQL instance. Idle connections (`Command=Sleep`) are excluded by default. Pass `include_idle: true` to show all |
| `mysql_explain` | Run `EXPLAIN` on a query and return the execution plan as structured rows. Optionally run `EXPLAIN ANALYZE` to include actual execution metrics (warning: executes the query) |
| `mysql_variables` | Run `SHOW GLOBAL VARIABLES` on a MySQL instance. Optionally filter by variable name using a `LIKE` pattern (e.g. `innodb%`) |
| `mysql_status` | Run `SHOW GLOBAL STATUS` on a MySQL instance. Optionally filter by variable name using a `LIKE` pattern (e.g. `Innodb%`, `Threads%`) |
| `mysql_innodb` | Run `SHOW ENGINE INNODB STATUS` and return parsed structured output: semaphores, latest deadlock (queries and victim), transactions, file I/O, log, buffer pool and row operations |
| `mysql_overflow` | Check AUTO_INCREMENT overflow risk for all tables in a database. Returns current value, max value, percentage used, and remaining capacity per column, sorted by percentage used descending |
| `mysql_performance` | Analyze MySQL configuration variables and return tuning recommendations with status (`ok` / `warning`). Checks: InnoDB buffer pool chunk size alignment and chunk size ratio (optimal: 2–5% of buffer pool) |
| `mysql_health_check` | Run health checks on a MySQL instance and return key metrics with status (`ok` / `warning` / `critical`). Checks: InnoDB buffer pool hit rate, buffer pool size vs available RAM, thread cache hit rate, thread cache ratio, temporary tables on disk, InnoDB history list length, max connections usage, InnoDB dirty pages ratio, open files utilization, flushing logs ratio, sort merge passes ratio, and InnoDB redo log fill time |
| `mysql_schema_check` | Run schema-level checks on a MySQL instance with status (`ok` / `warning`). Checks: deprecated table engine (MyISAM) and missing primary keys |

## Credentials

Tools that connect directly to MySQL read credentials from `~/.my.cnf`. Each RDS instance must have its own section named after the instance identifier:

```ini
[com-prd-mysql-general-node01]
host=com-prd-mysql-general-node01.xxxxxxxxxxxx.eu-west-1.rds.amazonaws.com
user=your_user
password=your_password
port=3306
```

The section name must match exactly the `db_instance_identifier` used in the tool call.

### Connecting via SSH Tunnel

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
