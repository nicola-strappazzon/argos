package mysqlconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nicola-strappazzon/argos/internal/cnf"
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
	keys, err := cnf.Load(path, section)
	if err != nil {
		return nil, err
	}

	creds := &Credentials{Port: 3306}
	ssh := &SSHConfig{Port: 22}

	creds.Host = keys["host"]
	creds.User = keys["user"]
	creds.Password = keys["password"]
	if v, ok := keys["port"]; ok {
		fmt.Sscanf(v, "%d", &creds.Port)
	}

	ssh.Host = keys["ssh_host"]
	ssh.User = keys["ssh_user"]
	ssh.Key = keys["ssh_key"]
	if v, ok := keys["ssh_port"]; ok {
		fmt.Sscanf(v, "%d", &ssh.Port)
	}

	if ssh.Host != "" {
		creds.SSH = ssh
	}

	if creds.User == "" {
		return nil, fmt.Errorf("section [%s] not found in ~/.my.cnf", section)
	}

	return creds, nil
}
