package shared

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stretchr/testify/require"
)

func TestTLSWithAdditonalCA(t *testing.T) {
	ca, err := certgen.GenerateCA()
	require.NoError(t, err, "failed to generate test CA")

	caDir := t.TempDir()
	filePath := path.Join(caDir, "cert.pem")
	err = os.WriteFile(filePath, ca.CertPEM(), 0644)
	require.NoError(t, err, "failed to write test CA to file")

	testServerCalled := false
	tlsServ := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testServerCalled = true
		w.WriteHeader(200)
	}))

	tlsServ.TLS = &tls.Config{
		Certificates: []tls.Certificate{generateTestServerCert(t, ca)},
	}

	tlsServ.StartTLS()
	defer tlsServ.Close()

	tlsConf, err := TLSWithAdditionalCAs(filePath)
	require.NoError(t, err, "failed to create tls config")

	httpClient := http.Client{Transport: &http.Transport{
		TLSClientConfig: tlsConf,
	}}

	err = retry.WithRetry(
		// there's a chance the first call fails on tests depending on
		// server startup timing
		func() error {
			_, err := httpClient.Get(tlsServ.URL)
			return err
		},
		retry.Tries(3),
		retry.BetweenAttempts(func(_ int) {
			time.Sleep(1 * time.Second)
		}),
	)

	require.NoError(t, err, "expected HTTP call to test server to succeed")
	require.True(t, testServerCalled, "expected test server to be called succesfully")
}

func generateTestServerCert(t *testing.T, ca mtls.CA) tls.Certificate {
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	certKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "failed to generate test server TLS key")

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.Certificate(), certKey.Public(), ca.PrivateKey())
	require.NoError(t, err, "failed to generate test server TLS cert")

	certPem := &bytes.Buffer{}
	err = pem.Encode(certPem, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	require.NoError(t, err, "failed to encode test server TLS cert to pem")

	keyDER, err := x509.MarshalPKCS8PrivateKey(certKey)
	require.NoError(t, err, "failed to marshal test server TLS key")
	keyPem := &bytes.Buffer{}
	err = pem.Encode(keyPem, &pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	require.NoError(t, err, "failed to encode test server TLS key to pem")

	cert, err := tls.X509KeyPair(certPem.Bytes(), keyPem.Bytes())
	require.NoError(t, err, "failed to create test server TLS key pair")
	return cert
}
