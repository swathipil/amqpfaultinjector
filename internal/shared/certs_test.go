package shared_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Azure/amqpfaultinjector/internal/shared"
	"github.com/stretchr/testify/require"
)

func TestLoadOrCreateCert(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "temp-cert-test-*")
	require.NoError(t, err)

	defer os.RemoveAll(tmpDir)

	certFile, keyFile, cert, err := shared.LoadOrCreateCert(tmpDir)
	require.NoError(t, err)
	require.FileExists(t, certFile)
	require.FileExists(t, keyFile)
	require.NotEmpty(t, cert)

	require.True(t, strings.HasSuffix(certFile, "/server.crt"))
	require.True(t, strings.HasSuffix(keyFile, "/server.key"))

	beforeCertStat, err := os.Stat(certFile)
	require.NoError(t, err)

	beforeKeyStat, err := os.Stat(keyFile)
	require.NoError(t, err)

	// now, "regen" again - it should just use the existing key/cert and not create a new one.
	certFile, keyFile, cert, err = shared.LoadOrCreateCert(tmpDir)
	require.NoError(t, err)
	require.FileExists(t, certFile)
	require.FileExists(t, keyFile)
	require.NotEmpty(t, cert)

	afterCertStat, err := os.Stat(certFile)
	require.NoError(t, err)

	afterKeyStat, err := os.Stat(keyFile)
	require.NoError(t, err)

	require.Equal(t, beforeCertStat.ModTime(), afterCertStat.ModTime())
	require.Equal(t, beforeKeyStat.ModTime(), afterKeyStat.ModTime())
}
