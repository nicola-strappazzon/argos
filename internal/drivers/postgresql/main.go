package postgresql

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/nicola-strappazzon/argos/internal/config/psql"
)

// Connect opens a PostgreSQL connection for the given RDS instance identifier.
// Credentials are read from ~/.pgpass matching the instance identifier against the hostname.
func Connect(instanceID string) (*sql.DB, error) {
	creds, err := psqlconfig.Load(instanceID)
	if err != nil {
		return nil, err
	}
	return open(creds.Host, creds.Port, creds.User, creds.Password, creds.Database, instanceID)
}

// ConnectDB opens a PostgreSQL connection to a specific database on the given instance.
// Useful when querying objects that require a direct connection to the target database.
func ConnectDB(instanceID, database string) (*sql.DB, error) {
	creds, err := psqlconfig.Load(instanceID)
	if err != nil {
		return nil, err
	}
	return open(creds.Host, creds.Port, creds.User, creds.Password, database, instanceID)
}

func open(host string, port int, user, password, database, instanceID string) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
		host, port, user, password, database)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening postgresql connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connecting to %s: %w", instanceID, err)
	}

	return db, nil
}
