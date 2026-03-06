package mysql

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	gomysql "github.com/go-sql-driver/mysql"
	mysqlconfig "github.com/nicola-strappazzon/argos/internal/config/mysql"
	"golang.org/x/crypto/ssh"
)

var (
	sshMu      sync.Mutex
	sshTunnels = map[string]*ssh.Client{}
)

func ensureSSHTunnel(instanceID string, creds *mysqlconfig.Credentials) error {
	dialerName := "mysql+ssh+" + instanceID

	sshMu.Lock()
	defer sshMu.Unlock()

	if _, ok := sshTunnels[instanceID]; ok {
		return nil
	}

	client, err := newSSHClient(creds.SSH)
	if err != nil {
		return err
	}

	sshTunnels[instanceID] = client
	gomysql.RegisterDialContext(dialerName, func(ctx context.Context, addr string) (net.Conn, error) {
		return client.Dial("tcp", addr)
	})

	return nil
}

func newSSHClient(cfg *mysqlconfig.SSHConfig) (*ssh.Client, error) {
	keyPath := cfg.Key
	if strings.HasPrefix(keyPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("getting home directory: %w", err)
		}
		keyPath = filepath.Join(home, keyPath[2:])
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading ssh key %s: %w", keyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("parsing ssh key: %w", err)
	}

	config := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("connecting to ssh host %s: %w", addr, err)
	}

	return client, nil
}
