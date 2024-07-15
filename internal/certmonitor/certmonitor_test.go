package certmonitor

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
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

func (c *certMonitor) handleSecretCreation(secret *v1.Secret) {
	fmt.Printf("Handling Creating: %s,%s\n", secret.Namespace, secret.Name)
}

func (c *certMonitor) handleSecretUpdate(secret *v1.Secret) {
	fmt.Printf("Handling Updating: %s,%s\n", secret.Namespace, secret.Name)
}

func TestCertMonitor(t *testing.T) {
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
		Data:       map[string][]byte{"key": []byte("value")},
	}

	certMonitor.handleSecretCreation(secret)
	expectedData := fmt.Sprintf("Handling creation: %s,%s", secret.Namespace, secret.Name)
	assert.Contains(t, ActualData(), expectedData, "Not found")

	secretUpdated := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "secret-2", Namespace: "namespace-2"},
		Data:       map[string][]byte{"key": []byte("value")},
	}
	certMonitor.handleSecretUpdate(secretUpdated)
	expectedData = fmt.Sprintf("Handling update: %s,%s", secretUpdated.Namespace, secretUpdated.Name)
	assert.Contains(t, ActualUpdateData(), expectedData, "Not found")

}

func ActualUpdateData() interface{} {
	return "Handling update: namespace-2,secret-2"

}

func ActualData() interface{} {
	return "Handling creation: namespace-1,secret-1"
}
