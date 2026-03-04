package docdbconfig

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening ~/.docdb: %w", err)
	}
	defer f.Close()

	creds := &Credentials{Port: 27017}
	inSection := false
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inSection = line[1:len(line)-1] == section
			continue
		}

		if !inSection {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "host":
			creds.Host = value
		case "port":
			fmt.Sscanf(value, "%d", &creds.Port)
		case "user":
			creds.User = value
		case "password":
			creds.Password = value
		case "tls":
			creds.TLS = value == "true" || value == "1" || value == "yes"
		case "tls_ca_file":
			creds.TLSCAFile = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading ~/.docdb: %w", err)
	}

	if creds.User == "" {
		return nil, fmt.Errorf("section [%s] not found in ~/.docdb", section)
	}

	return creds, nil
}
