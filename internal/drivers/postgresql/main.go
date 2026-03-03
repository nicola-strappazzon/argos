package postgresql

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/nicola-strappazzon/argos/internal/psqlconfig"
)

// Connect opens a PostgreSQL connection for the given RDS instance identifier.
// Credentials are read from ~/.pgpass matching the instance identifier against the hostname.
func Connect(instanceID string) (*sql.DB, error) {
	creds, err := psqlconfig.Load(instanceID)
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
		creds.Host, creds.Port, creds.User, creds.Password, creds.Database)

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
