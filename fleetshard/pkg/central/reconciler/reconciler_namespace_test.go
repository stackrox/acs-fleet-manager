package reconciler

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"
	"testing"
)

func TestNamespaceReconciler(t *testing.T) {
	tests := []struct {
		name              string
		existingNamespace *corev1.Namespace
		wantErr           bool
		wantNamespace     *corev1.Namespace
		expectUpdate      bool
		expectCreate      bool
	}{
		{
			name:         "namespace should be created if it doesn't exist",
			expectCreate: true,
			wantNamespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: simpleManagedCentral.Metadata.Namespace,
					Labels: map[string]string{
						"app.kubernetes.io/instance":     "test-central",
						"app.kubernetes.io/managed-by":   "rhacs-fleetshard",
						"rhacs.redhat.com/instance-type": "standard",
						"rhacs.redhat.com/org-id":        "12345",
						"rhacs.redhat.com/tenant":        "cb45idheg5ip6dq1jo4g",
					},
					Annotations: map[string]string{
						"rhacs.redhat.com/org-name": "org-name",
						ovnACLLoggingAnnotationKey:  ovnACLLoggingAnnotationDefault,
					},
				},
			},
		},
		{
			name:         "namespace with wrong labels or annotations should be updated",
			expectUpdate: true,
			existingNamespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: simpleManagedCentral.Metadata.Namespace,
					Labels: map[string]string{
						"app.kubernetes.io/instance":     "wrong",
						"app.kubernetes.io/managed-by":   "wrong",
						"rhacs.redhat.com/instance-type": "wrong",
						"rhacs.redhat.com/org-id":        "wrong",
						"rhacs.redhat.com/tenant":        "wrong",
					},
					Annotations: map[string]string{
						"rhacs.redhat.com/org-name": "wrong",
						ovnACLLoggingAnnotationKey:  "{\"allow\": \"wrong\"}",
					},
				},
			},
			wantNamespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: simpleManagedCentral.Metadata.Namespace,
					Labels: map[string]string{
						"app.kubernetes.io/instance":     "test-central",
						"app.kubernetes.io/managed-by":   "rhacs-fleetshard",
						"rhacs.redhat.com/instance-type": "standard",
						"rhacs.redhat.com/org-id":        "12345",
						"rhacs.redhat.com/tenant":        "cb45idheg5ip6dq1jo4g",
					},
					Annotations: map[string]string{
						"rhacs.redhat.com/org-name": "org-name",
						ovnACLLoggingAnnotationKey:  ovnACLLoggingAnnotationDefault,
					},
				},
			},
		},
		{
			name: "extra labels/annotations should remain untouched",
			existingNamespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: simpleManagedCentral.Metadata.Namespace,
					Labels: map[string]string{
						"app.kubernetes.io/instance":     "test-central",
						"app.kubernetes.io/managed-by":   "rhacs-fleetshard",
						"rhacs.redhat.com/instance-type": "standard",
						"rhacs.redhat.com/org-id":        "12345",
						"rhacs.redhat.com/tenant":        "cb45idheg5ip6dq1jo4g",
						"extra":                          "extra",
					},
					Annotations: map[string]string{
						"rhacs.redhat.com/org-name": "org-name",
						ovnACLLoggingAnnotationKey:  ovnACLLoggingAnnotationDefault,
						"extra":                     "extra",
					},
				},
			},
			wantNamespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: simpleManagedCentral.Metadata.Namespace,
					Labels: map[string]string{
						"app.kubernetes.io/instance":     "test-central",
						"app.kubernetes.io/managed-by":   "rhacs-fleetshard",
						"rhacs.redhat.com/instance-type": "standard",
						"rhacs.redhat.com/org-id":        "12345",
						"rhacs.redhat.com/tenant":        "cb45idheg5ip6dq1jo4g",
						"extra":                          "extra",
					},
					Annotations: map[string]string{
						"rhacs.redhat.com/org-name": "org-name",
						ovnACLLoggingAnnotationKey:  ovnACLLoggingAnnotationDefault,
						"extra":                     "extra",
					},
				},
			},
		},
		{
			name: "namespace should not be updated if it's already correct",
			existingNamespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: simpleManagedCentral.Metadata.Namespace,
					Labels: map[string]string{
						"app.kubernetes.io/instance":     "test-central",
						"app.kubernetes.io/managed-by":   "rhacs-fleetshard",
						"rhacs.redhat.com/instance-type": "standard",
						"rhacs.redhat.com/org-id":        "12345",
						"rhacs.redhat.com/tenant":        "cb45idheg5ip6dq1jo4g",
					},
					Annotations: map[string]string{
						"rhacs.redhat.com/org-name": "org-name",
						ovnACLLoggingAnnotationKey:  ovnACLLoggingAnnotationDefault,
					},
				},
			},
			wantNamespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: simpleManagedCentral.Metadata.Namespace,
					Labels: map[string]string{
						"app.kubernetes.io/instance":     "test-central",
						"app.kubernetes.io/managed-by":   "rhacs-fleetshard",
						"rhacs.redhat.com/instance-type": "standard",
						"rhacs.redhat.com/org-id":        "12345",
						"rhacs.redhat.com/tenant":        "cb45idheg5ip6dq1jo4g",
					},
					Annotations: map[string]string{
						"rhacs.redhat.com/org-name": "org-name",
						ovnACLLoggingAnnotationKey:  ovnACLLoggingAnnotationDefault,
					},
				},
			},
		},
	}

	managedCentral := simpleManagedCentral

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx := context.Background()
			ctx = withManagedCentral(ctx, managedCentral)

			var existingObjects []runtime.Object
			if tt.existingNamespace != nil {
				existingObjects = append(existingObjects, tt.existingNamespace)
			}

			fakeClientSet := fake.NewSimpleClientset(existingObjects...)

			updateCount := 0
			createCount := 0

			fakeClientSet.PrependReactor("update", "namespaces", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
				updateCount++
				return false, nil, nil
			})

			fakeClientSet.PrependReactor("create", "namespaces", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
				createCount++
				return false, nil, nil
			})

			ctx, err := newNamespaceReconciler(fakeClientSet).ensurePresent(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				got, err := fakeClientSet.CoreV1().Namespaces().Get(ctx, simpleManagedCentral.Metadata.Namespace, metav1.GetOptions{})
				require.NoError(t, err)
				assert.Equal(t, tt.wantNamespace.Name, got.Name)
				assert.Equal(t, tt.wantNamespace.Labels, got.Labels)
				assert.Equal(t, tt.wantNamespace.Annotations, got.Annotations)
				if tt.expectUpdate {
					assert.Equal(t, 1, updateCount, "update should be called")
				} else {
					assert.Equal(t, 0, updateCount, "update should not be called")
				}
				if tt.expectCreate {
					assert.Equal(t, 1, createCount, "create should be called")
				} else {
					assert.Equal(t, 0, createCount, "create should not be called")
				}
			}
		})
	}
}
