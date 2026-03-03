package psqlconfig

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Credentials struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
}

// Load reads credentials for the given instance from ~/.pgpass.
// It matches lines where the hostname equals the instanceID or starts with instanceID+"."
// (i.e. the RDS endpoint prefix matches the instance identifier).
// The ~/.pgpass format is: hostname:port:database:username:password
func Load(instanceID string) (*Credentials, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}

	path := filepath.Join(home, ".pgpass")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening ~/.pgpass: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.SplitN(line, ":", 5)
		if len(fields) != 5 {
			continue
		}

		hostname := fields[0]
		if hostname != instanceID && !strings.HasPrefix(hostname, instanceID+".") {
			continue
		}

		creds := &Credentials{
			Host:     hostname,
			Port:     5432,
			Database: fields[2],
			User:     fields[3],
			Password: fields[4],
		}

		if fields[1] != "*" {
			fmt.Sscanf(fields[1], "%d", &creds.Port)
		}

		if creds.Database == "*" {
			creds.Database = "postgres"
		}

		return creds, nil
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading ~/.pgpass: %w", err)
	}

	return nil, fmt.Errorf("no entry for [%s] found in ~/.pgpass", instanceID)
}
