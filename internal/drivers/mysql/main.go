package mysql

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/nicola-strappazzon/argos/internal/mysqlconfig"
)

// Connect opens a MySQL connection for the given RDS instance identifier.
// Credentials are read from ~/.my.cnf using the instance ID as the section name.
func Connect(instanceID string) (*sql.DB, error) {
	creds, err := mysqlconfig.Load(instanceID)
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/", creds.User, creds.Password, creds.Host, creds.Port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening mysql connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connecting to %s: %w", instanceID, err)
	}

	return db, nil
}
