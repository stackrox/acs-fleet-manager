package reconciler

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	fake2 "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
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
		}, {
			name: "namespace should not be updated if it being deleted",
			existingNamespace: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:              simpleManagedCentral.Metadata.Namespace,
					DeletionTimestamp: &metav1.Time{},
					Finalizers:        []string{"foo"},
				},
			},
			wantErr: true,
		},
	}

	managedCentral := simpleManagedCentral

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx := withManagedCentral(context.Background(), managedCentral)

			var existingObjects []ctrlClient.Object
			if tt.existingNamespace != nil {
				existingObjects = append(existingObjects, tt.existingNamespace)
			}

			updateCount := 0
			createCount := 0

			fakeClient := fake2.NewClientBuilder().
				WithInterceptorFuncs(interceptor.Funcs{
					Create: func(ctx context.Context, client ctrlClient.WithWatch, obj ctrlClient.Object, opts ...ctrlClient.CreateOption) error {
						createCount++
						return client.Create(ctx, obj, opts...)
					},
					Update: func(ctx context.Context, client ctrlClient.WithWatch, obj ctrlClient.Object, opts ...ctrlClient.UpdateOption) error {
						updateCount++
						return client.Update(ctx, obj, opts...)
					},
				}).WithObjects(existingObjects...).
				Build()

			ctx, err := newNamespaceReconciler(fakeClient).ensurePresent(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				var got corev1.Namespace
				err := fakeClient.Get(ctx, ctrlClient.ObjectKey{Name: simpleManagedCentral.Metadata.Namespace}, &got)
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
