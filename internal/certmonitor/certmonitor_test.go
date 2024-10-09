package certmonitor

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetshardmetrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"math/big"
	"testing"
	"time"
)

// fakeNameSpaceGetter struct is mock implementation for test
type fakeNamespaceGetter struct {
	namespaces map[string]v1.Namespace
}

// Get func returns/gets a namespace by name
func (f *fakeNamespaceGetter) Get(name string) (*v1.Namespace, error) {
	ns, ok := f.namespaces[name]
	if !ok {
		return nil, fmt.Errorf("namespace %q not found", name)
	}
	return &ns, nil
}

// newFakeNamespaceGetter func creates new fakeNameSpaceGetter
func newFakeNamespaceGetter(namespaces []v1.Namespace) *fakeNamespaceGetter {
	f := fakeNamespaceGetter{namespaces: make(map[string]v1.Namespace)}
	for _, ns := range namespaces {
		f.namespaces[ns.Name] = ns
	}
	return &f
}

// TestCertMonitor_secretMatches func tests the secretMatches method in certmonitor.go
func TestCertMonitor_secretMatches(t *testing.T) {
	tests := []struct {
		name       string
		secret     v1.Secret
		monitor    MonitorConfig
		want       bool
		namespaces []v1.Namespace
	}{
		{
			name: "should match on namespace and secret name",

			secret: v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "secret-1", Namespace: "namespace-1"}}, // pragma: allowlist secret
			monitor: MonitorConfig{
				Namespace: SelectorConfig{Name: "namespace-1"},
				Secret:    SelectorConfig{Name: "secret-1"}, // pragma: allowlist secret
			},
			want: true,
		}, {
			name:   "mismatch on namespace name",
			secret: v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "secret-1", Namespace: "namespace-1"}}, // pragma: allowlist secret
			monitor: MonitorConfig{
				Namespace: SelectorConfig{Name: "foo"},
				Secret:    SelectorConfig{Name: "secret-1"}, // pragma: allowlist secret
			}, // pragma: allowlist secret
			want: false,
		}, {
			name:   "mismatch on secret name",
			secret: v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "secret-1", Namespace: "namespace-1"}}, // pragma: allowlist secret
			monitor: MonitorConfig{
				Namespace: SelectorConfig{Name: "namespace-1"},
				Secret:    SelectorConfig{Name: "bar"}, // pragma: allowlist secret
			},
			want: false,
		}, {
			name: "match on namespace name and secret label",
			secret: v1.Secret{ // pragma: allowlist secret
				ObjectMeta: metav1.ObjectMeta{ // pragma: allowlist secret
					Name:      "secret-1", // pragma: allowlist secret
					Namespace: "namespace-1",
					Labels: map[string]string{
						"foo": "bar",
					},
				},
			},
			monitor: MonitorConfig{
				Namespace: SelectorConfig{Name: "namespace-1"}, // pragma: allowlist secret
				Secret: SelectorConfig{ // pragma: allowlist secret
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"foo": "bar", // pragma: allowlist secret
						},
					},
				},
			},
			namespaces: []v1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace-1",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
			want: true,
		}, {
			name: "match on namespace label and secret name",
			secret: v1.Secret{ // pragma: allowlist secret
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-1", // pragma: allowlist secret
					Namespace: "namespace-1",
				},
			},
			monitor: MonitorConfig{
				Namespace: SelectorConfig{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"foo": "bar",
						},
					},
				},
				Secret: SelectorConfig{ // pragma: allowlist secret
					Name: "secret-1",
				},
			},
			namespaces: []v1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace-1",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
			want: true,
		}, {
			name: "match on both namespace label and secret label",
			secret: v1.Secret{ // pragma: allowlist secret
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-1",
					Namespace: "namespace-1",
					Labels: map[string]string{
						"foo": "bar",
					},
				},
			},
			monitor: MonitorConfig{
				Namespace: SelectorConfig{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"foo": "bar",
						},
					},
				},
				Secret: SelectorConfig{ // pragma: allowlist secret
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
			namespaces: []v1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace-1",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
			want: true,
		}, {
			name: "mismatch on both namespace label and secret label",
			secret: v1.Secret{ // pragma: allowlist secret
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-1",
					Namespace: "namespace-1",
					Labels: map[string]string{
						"foo": "bar",
					},
				},
			},
			monitor: MonitorConfig{
				Namespace: SelectorConfig{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"foo": "qux",
						},
					},
				},
				Secret: SelectorConfig{ // pragma: allowlist secret
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"foo": "qux",
						},
					},
				},
			},
			namespaces: []v1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace-1",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
			want: false,
		}, {
			name: "mismatch on namespace name and secret label",
			secret: v1.Secret{ // pragma: allowlist secret
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-1",
					Namespace: "namespace-1",
					Labels: map[string]string{
						"foo": "bar",
					},
				},
			},
			monitor: MonitorConfig{
				Namespace: SelectorConfig{Name: "namespace-2"},
				Secret: SelectorConfig{ // pragma: allowlist secret
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"foo": "qux",
						},
					},
				},
			},
			namespaces: []v1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace-1",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
			want: false,
		}, {
			name: "mismatch on namespace label and secret name",
			secret: v1.Secret{ // pragma: allowlist secret
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-1",
					Namespace: "namespace-1",
				},
			},
			monitor: MonitorConfig{
				Namespace: SelectorConfig{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"foo": "qux",
						},
					},
				},
				Secret: SelectorConfig{ // pragma: allowlist secret
					Name: "secret-2",
				},
			},
			namespaces: []v1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace-1",
						Labels: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			certMonitor := &certMonitor{
				namespaceGetter: newFakeNamespaceGetter(tt.namespaces),
			}
			got := certMonitor.secretMatches(&tt.secret, tt.monitor)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestCertMonitor func tests certificates event handlers + prometheus metrics handling
func TestCertMonitor(t *testing.T) {
	fleetshardmetrics.MetricsInstance().CertificatesExpiry.Reset()

	namespaces := []v1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace-1",
				Labels: map[string]string{
					"foo": "bar"},
			},
		},
	}
	certMonitor := &certMonitor{
		namespaceGetter: newFakeNamespaceGetter(namespaces),
		metrics:         fleetshardmetrics.MetricsInstance(),
		config: &Config{
			Monitors: []MonitorConfig{
				{
					Namespace: SelectorConfig{
						Name: "namespace-1",
					},
					Secret: SelectorConfig{ // pragma: allowlist secret
						Name: "secret-1",
					},
				},
			},
		},
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

	mockNamespace := &v1.Namespace{ // pragma: allowlist secret
		ObjectMeta: metav1.ObjectMeta{Namespace: "namespace-1", Name: "secret-1"},
	}

	expirationUnix := float64(expirytime.Unix())
	certMonitor.handleSecretCreation(secret)
	verifyPrometheusMetric(t, "namespace-1", "secret-1", "tls.crt", expirationUnix)

	updatedUnix := float64(newExpiryTime.Unix())
	certMonitor.handleSecretUpdate(secret, secretUpdated)
	verifyPrometheusMetric(t, "namespace-1", "secret-1", "tls-1.crt", updatedUnix)

	certMonitor.handleSecretDeletion(secretUpdated)
	verifyPrometheusMetricDelete(t, "namespace-1", "secret-1", "tls.crt")

	certMonitor.handleNamespaceDeletion(mockNamespace)
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
	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytesDER})
	return []byte(pemCert)
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
