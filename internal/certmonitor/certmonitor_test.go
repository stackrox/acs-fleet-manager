package certmonitor

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

type fakeNamespaceGetter struct {
	namespaces map[string]v1.Namespace
}

func (f *fakeNamespaceGetter) Get(name string) (*v1.Namespace, error) {
	ns, ok := f.namespaces[name]
	if !ok {
		return nil, fmt.Errorf("namespace %q not found", name)
	}
	return &ns, nil
}

func newFakeNamespaceGetter(namespaces []v1.Namespace) *fakeNamespaceGetter {
	f := fakeNamespaceGetter{namespaces: make(map[string]v1.Namespace)}
	for _, ns := range namespaces {
		f.namespaces[ns.Name] = ns
	}
	return &f
}

func TestCertMonitor_secretMatches(t *testing.T) {
	tests := []struct {
		name       string
		secret     v1.Secret
		monitor    MonitorConfig
		want       bool
		namespaces []v1.Namespace
	}{
		{
			name:   "should match on namespace and secret name",
			secret: v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "secret-1", Namespace: "namespace-1"}},
			monitor: MonitorConfig{
				Namespace: SelectorConfig{Name: "namespace-1"},
				Secret:    SelectorConfig{Name: "secret-1"},
			},
			want: true,
		}, {
			name:   "mismatch on namespace name",
			secret: v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "secret-1", Namespace: "namespace-1"}},
			monitor: MonitorConfig{
				Namespace: SelectorConfig{Name: "foo"},
				Secret:    SelectorConfig{Name: "secret-1"},
			},
			want: false,
		}, {
			name:   "mismatch on secret name",
			secret: v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "secret-1", Namespace: "namespace-1"}},
			monitor: MonitorConfig{
				Namespace: SelectorConfig{Name: "namespace-1"},
				Secret:    SelectorConfig{Name: "bar"},
			},
			want: false,
		}, {
			name: "match on namespace name and secret label",
			secret: v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-1",
					Namespace: "namespace-1",
					Labels: map[string]string{
						"foo": "bar",
					},
				},
			},
			monitor: MonitorConfig{
				Namespace: SelectorConfig{Name: "namespace-1"},
				Secret: SelectorConfig{
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
			name: "match on namespace label and secret name",
			secret: v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret-1",
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
				Secret: SelectorConfig{
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
			secret: v1.Secret{
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
				Secret: SelectorConfig{
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
			secret: v1.Secret{
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
				Secret: SelectorConfig{
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
			secret: v1.Secret{
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
				Secret: SelectorConfig{
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
			secret: v1.Secret{
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
				Secret: SelectorConfig{
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
				namespaceLister: newFakeNamespaceGetter(tt.namespaces),
			}
			got := certMonitor.secretMatches(&tt.secret, tt.monitor)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCertMonitor(t *testing.T) {
	cert := "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUVuakNDQXdhZ0F3SUJBZ0lRSEdGOTR4QVIxMkhsS3ZaZ1A5Vng1REFOQmdrcWhraUc5dzBCQVFzRkFEQ0IKdVRFZU1Cd0dBMVVFQ2hNVmJXdGpaWEowSUdSbGRtVnNiM0J0Wlc1MElFTkJNVWN3UlFZRFZRUUxERDVoYTJGdApaWEpwWTBCaGEyRnRaWEpwWXkxMGFHbHVhM0JoWkhReE5ITm5aVzR5YVM1eVpXMXZkR1V1WTNOaUlDaEJiV2x1CllTQkxZVzFsY21saktURk9NRXdHQTFVRUF3eEZiV3RqWlhKMElHRnJZVzFsY21salFHRnJZVzFsY21sakxYUm8KYVc1cmNHRmtkREUwYzJkbGJqSnBMbkpsYlc5MFpTNWpjMklnS0VGdGFXNWhJRXRoYldWeWFXTXBNQjRYRFRJMApNRFl5TnpBNU1UTTFObG9YRFRJMk1Ea3lOekE1TVRNMU5sb3djakVuTUNVR0ExVUVDaE1lYld0alpYSjBJR1JsCmRtVnNiM0J0Wlc1MElHTmxjblJwWm1sallYUmxNVWN3UlFZRFZRUUxERDVoYTJGdFpYSnBZMEJoYTJGdFpYSnAKWXkxMGFHbHVhM0JoWkhReE5ITm5aVzR5YVM1eVpXMXZkR1V1WTNOaUlDaEJiV2x1WVNCTFlXMWxjbWxqS1RDQwpBU0l3RFFZSktvWklodmNOQVFFQkJRQURnZ0VQQURDQ0FRb0NnZ0VCQUx6MHJUZVdXaWJWMmQ1TGFMbVhNL0FkCm1WNTZNR2t4WnRrcXhhait6eWpUSjYySUd3cVFIMDBNYjdLY0wyYm1GK0RHR096M1ZLWkhERk5QV3NBU3FOcFoKeWlmOFNKRFVVTkFGd1lqd0ZoQS9EWjA2dnlsMjNlYlJXaGZJVW1uNjhHWFMxNWZudGZYa3pQSFQrcmFocmRMOApnUmtHbmNyQklSZm9ab3l1WFJ5VDlXYUY2R0ZuRnIzY3I0VjQrSUUxM1cyS1JFVVJlUDdxU2NwK2dmQTl3QXcrCmhYaWo2OVFxWmJxOTM1ZEZ0TVhCeW5XV1RBWHJ0SDl6TkVWTDRuM05Tc29CM01KOHQxbjlSRmVoL0l6UVBLRFoKaDNVZXJPMml6eFVGOU40MzUvNStxclJ0WjJFOGtGVU5YV3lZUWd4Sm04dU5OL3FJZGMzWEp1cXducXl4N2IwQwpBd0VBQWFOb01HWXdEZ1lEVlIwUEFRSC9CQVFEQWdXZ01CTUdBMVVkSlFRTU1Bb0dDQ3NHQVFVRkJ3TUVNQjhHCkExVWRJd1FZTUJhQUZDdE9qTUQyZVZWeGt2Q0YvT2xqQSsrV1FYRDVNQjRHQTFVZEVRUVhNQldCRTJGcllXMWwKY21salFISmxaR2hoZEM1amIyMHdEUVlKS29aSWh2Y05BUUVMQlFBRGdnR0JBR2dLZ1Z0NTFEMmpaS01QQWxUZQpUbVRZOUNVQW8veUtFaEpaeVNuZTY3T3pSNSsvRVhsRWpiWkJkOHJyaC9xcW9uaER4ZmFFM1BXSkdVclZMZXd6Ck1KTE5QZFVtZmRYbXcwdEVIU1VHOVRBSjM1Z2lMZUtpbjJQTytMZk42Z1pyd1VkWWhvSWxySjBFSFgyam1sbWQKUkJoZjhkaTlSelZWSjJFUkpyL3VTNmFCWVB5WERUd1I0Nk1WZ3FaZGFXb2ZuVlhBbUNud3BRM0J5N2tOeTVSZgpMZTN6K3RhZVQ1cG5sR1VoZmltTU9sc1pEanFWajNJaHR0bExBVzlZbGJ6NFlxcmgxbVJjNG5kY2xVbzFXOXdDCjJHU3FhdWNLdUdNQ0p2VmJlUXRMQzBwWUJXdTFQT2N1QmVpM0xzY2U4VHYyQ0VqOHF4c3FtSjdWak0xM0g3Y1kKMWxTSytLb3oycHB3ZWd3WUc2TTdjNUdxNk5XbWF3czg2dTExTFRnQ0phUDFoZTJnWjZ2eTR2eFF3eVcvbWpkTwpWeEx0NHN1UUl3bW4rcCtuK0phTFdBck9oRFFNZEpKSTdtSit5d0lONk1yekc4TXFaK0Y1QzlicUZYVSs4WWFQCm1TUDJPQ3p2YmJESUkyekh0d1kyQ3doZ2FLeHJoV3ZEQjE4TzlSck4va3l2K1E9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="

	now1 := time.Now()
	expirytime := now1.Add(1 * time.Minute)
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
		namespaceLister: newFakeNamespaceGetter(namespaces),
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret-1", Namespace: "namespace-1"},
		Data:       map[string][]byte{"tls.crt": []byte(cert)},
	}

	certMonitor.handleTestSecretCreation(secret)
	expectedData := fmt.Sprintf("Handling creation: %s,%s", secret.Namespace, secret.Name)
	assert.Contains(t, ActualData(), expectedData, "Not found")

	expirationUnix := float64(expirytime.Unix())

	verifyPrometheusMetric(t, "namespace-1", "secret-1", "tls.crt", expirationUnix)

	secretUpdated := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret-2", Namespace: "namespace-2"},
		Data:       map[string][]byte{"tls.crt": []byte(cert)},
	}
	certMonitor.handleTestSecretUpdate(secret, secretUpdated)
	expectedData = fmt.Sprintf("Handling update: %s,%s", secretUpdated.Namespace, secretUpdated.Name)
	assert.Contains(t, ActualUpdateData(), expectedData, "Not found")

	certMonitor.handleTestSecretDeletion(secret)
	expectedData = fmt.Sprintf("Handling delete: %s,%s", secret.Namespace, secret.Name)
	assert.Contains(t, ActualDeleteData(), expectedData, "Not found")

}

func verifyPrometheusMetric(t *testing.T, namespace, secret, dataKey string, expectedValue float64) {
	actualValue := testutil.ToFloat64(certificatesExpiry.WithLabelValues(namespace, secret, dataKey))
	assert.Equal(t, expectedValue, actualValue, "Value does nt match")
}

func ActualDeleteData() interface{} {
	return "Handling delete: namespace-1,secret-1"
}

func ActualUpdateData() interface{} {
	return "Handling update: namespace-2,secret-2"

}

func ActualData() interface{} {
	return "Handling creation: namespace-1,secret-1"
}
