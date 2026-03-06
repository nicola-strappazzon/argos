# PostgreSQL Tools

## Tools

| Tool | Description |
|---|---|
| `postgresql_ping` | Test the connection to a PostgreSQL instance. Returns success status and round-trip latency in milliseconds |
| `postgresql_databases` | List databases on a PostgreSQL instance with their size (MB), encoding, collation, owner and connection limit |
| `postgresql_tables` | List tables within a PostgreSQL database with detailed info: schema, owner, access method, estimated row count, dead tuples, size (data/index/total), comment and last vacuum/analyze timestamps |

## Credentials

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
