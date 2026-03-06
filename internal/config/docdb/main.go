package docdbconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nicola-strappazzon/argos/internal/cnf"
)

type Credentials struct {
	Host      string
	Port      int
	User      string
	Password  string
	TLS       bool
	TLSCAFile string
}

// Load reads credentials for the given section name from ~/.docdb.
// The section name is typically the DocumentDB instance identifier.
func Load(section string) (*Credentials, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}

	path := filepath.Join(home, ".docdb")
	keys, err := cnf.Load(path, section)
	if err != nil {
		return nil, err
	}

	creds := &Credentials{Port: 27017}

	creds.Host = keys["host"]
	creds.User = keys["user"]
	creds.Password = keys["password"]
	creds.TLSCAFile = keys["tls_ca_file"]
	if v, ok := keys["port"]; ok {
		fmt.Sscanf(v, "%d", &creds.Port)
	}
	if v := keys["tls"]; v == "true" || v == "1" || v == "yes" {
		creds.TLS = true
	}

	if creds.User == "" {
		return nil, fmt.Errorf("section [%s] not found in ~/.docdb", section)
	}

	return creds, nil
}
