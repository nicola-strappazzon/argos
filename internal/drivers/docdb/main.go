package docdb

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	docdbconfig "github.com/nicola-strappazzon/argos/internal/config/docdb"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Connect opens a MongoDB/DocumentDB connection for the given instance identifier.
// Credentials are read from ~/.docdb using the instance ID as the section name.
func Connect(instanceID string) (*mongo.Client, error) {
	creds, err := docdbconfig.Load(instanceID)
	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("mongodb://%s:%s@%s:%d/?directConnection=true",
		creds.User, creds.Password, creds.Host, creds.Port)

	clientOpts := options.Client().ApplyURI(uri)

	if creds.TLS {
		tlsCfg := &tls.Config{}

		if creds.TLSCAFile != "" {
			caCert, err := os.ReadFile(creds.TLSCAFile)
			if err != nil {
				return nil, fmt.Errorf("reading TLS CA file: %w", err)
			}
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("parsing TLS CA certificate from %s", creds.TLSCAFile)
			}
			tlsCfg.RootCAs = pool
		}

		clientOpts.SetTLSConfig(tlsCfg)
	}

	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", instanceID, err)
	}

	return client, nil
}
