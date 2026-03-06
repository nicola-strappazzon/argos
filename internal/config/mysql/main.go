package mysqlconfig

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SSHConfig struct {
	Host string
	Port int
	User string
	Key  string
}

type Credentials struct {
	Host     string
	Port     int
	User     string
	Password string
	SSH      *SSHConfig
}

// Load reads credentials for the given section name from ~/.my.cnf.
// The section name is typically the RDS instance identifier.
func Load(section string) (*Credentials, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}

	path := filepath.Join(home, ".my.cnf")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening ~/.my.cnf: %w", err)
	}
	defer f.Close()

	creds := &Credentials{Port: 3306}
	ssh := &SSHConfig{Port: 22}
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
		case "ssh_host":
			ssh.Host = value
		case "ssh_port":
			fmt.Sscanf(value, "%d", &ssh.Port)
		case "ssh_user":
			ssh.User = value
		case "ssh_key":
			ssh.Key = value
		}
	}

	if ssh.Host != "" {
		creds.SSH = ssh
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading ~/.my.cnf: %w", err)
	}

	if creds.User == "" {
		return nil, fmt.Errorf("section [%s] not found in ~/.my.cnf", section)
	}

	return creds, nil
}
