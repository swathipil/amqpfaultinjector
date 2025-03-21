package shared

import (
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"

	"github.com/madflojo/testcerts"
)

var emptyCert = tls.Certificate{}

// LoadOrCreateCert will create a server.key and a server.cert in the specified directory. If the files already
// exist it will load them, instead.
func LoadOrCreateCert(dir string) (certFile string, keyFile string, cert tls.Certificate, err error) {
	certFile = filepath.Join(dir, "server.crt")
	keyFile = filepath.Join(dir, "server.key")

	if err := os.MkdirAll(dir, 0600); err != nil {
		return "", "", emptyCert, fmt.Errorf("failed to create cert directory: %w", err)
	}

	_, certErr := os.Stat(certFile)
	_, keyErr := os.Stat(keyFile)

	if os.IsNotExist(certErr) || os.IsNotExist(keyErr) {
		if err := testcerts.GenerateCertsToFile(certFile, keyFile); err != nil {
			return "", "", emptyCert, err
		}
	}

	cert, err = tls.LoadX509KeyPair(certFile, keyFile)

	if err != nil {
		return "", "", emptyCert, fmt.Errorf("failed to load cert: %w", err)
	}

	return
}
