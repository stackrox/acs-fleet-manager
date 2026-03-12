package certmonitor

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetshardmetrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestCertMonitor_Secret tests that secret event handlers correctly populate prometheus metrics
func TestCertMonitor_Secret(t *testing.T) {
	fleetshardmetrics.MetricsInstance().CertificatesExpiry.Reset()

	certMonitor := &CertMonitor{
		metrics: fleetshardmetrics.MetricsInstance(),
	}
	now1 := time.Now().UTC()
	expirytime := now1.Add(1 * time.Hour)
	newExpiryTime := now1.Add(20 * time.Hour)

	secret := &v1.Secret{ // pragma: allowlist secret
		ObjectMeta: metav1.ObjectMeta{Namespace: "namespace-1", Name: "secret-1"},
		Data:       map[string][]byte{"tls.crt": generateCertWithExpiration(t, expirytime)},
	}

	secretUpdated := &v1.Secret{ // pragma: allowlist secret
		ObjectMeta: metav1.ObjectMeta{Namespace: "namespace-1", Name: "secret-1"},
		Data:       map[string][]byte{"tls-1.crt": generateCertWithExpiration(t, newExpiryTime)},
	}

	expirationUnix := float64(expirytime.Unix())
	certMonitor.handleSecretCreation(secret)
	verifyPrometheusMetric(t, "namespace-1", "secret-1", "tls.crt", expirationUnix)

	updatedUnix := float64(newExpiryTime.Unix())
	certMonitor.handleSecretUpdate(secret, secretUpdated)
	verifyPrometheusMetric(t, "namespace-1", "secret-1", "tls-1.crt", updatedUnix)

	certMonitor.handleSecretDeletion(secretUpdated)
	verifyPrometheusMetricDelete(t, "namespace-1", "secret-1", "tls.crt")
}

// generateCertWithExpiration func generates a pem-encoded certificate
func generateCertWithExpiration(t *testing.T, expiry time.Time) []byte {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	b, err := rand.Int(rand.Reader, big.NewInt(1000000))
	require.NoError(t, err)
	cert := &x509.Certificate{
		NotBefore:    time.Now(),
		NotAfter:     expiry,
		SerialNumber: b,
	}
	certBytesDER, err := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytesDER})
}

// verifyPrometheusMetric func verifies if the promethues metric matches the expected value (create + update handle)
func verifyPrometheusMetric(t *testing.T, namespace, secret, data_key string, expectedValue float64) {
	actualValue := testutil.ToFloat64(fleetshardmetrics.MetricsInstance().CertificatesExpiry.WithLabelValues(namespace, secret, data_key))
	assert.Equal(t, expectedValue, actualValue, "Value does not match")
}

// verifyPrometheusMetricDelete func verifies that the prometheus metric has actually been deleted (delete handle)
func verifyPrometheusMetricDelete(t *testing.T, namespace, secret, dataKey string) {
	metric, err := fleetshardmetrics.MetricsInstance().CertificatesExpiry.GetMetricWithLabelValues(namespace, secret, dataKey)
	require.NoError(t, err)
	require.Equal(t, float64(0), testutil.ToFloat64(metric))
}
