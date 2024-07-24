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

func Test_pullSecretReconciler(t *testing.T) {
	tests := []struct {
		name             string
		dockerConfigJson string
		existingSecret   *corev1.Secret
		wantErr          bool
		want             *corev1.Secret
		expectUpdate     bool
		expectCreate     bool
		expectDelete     bool
	}{
		{
			name:             "secret should be created if it doesn't exist and dockerConfigJson is not empty",
			expectCreate:     true,
			dockerConfigJson: `foo`,
			want: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tenantImagePullSecretName,
					Namespace: "test-namespace",
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`foo`), // pragma: allowlist secret
				},
			},
		}, {
			name:             "secret should be updated if it exists and dockerConfigJson is not empty and different",
			expectUpdate:     true,
			dockerConfigJson: `foo`,
			existingSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tenantImagePullSecretName,
					Namespace: "test-namespace",
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`bar`), // pragma: allowlist secret
				},
			},
			want: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tenantImagePullSecretName,
					Namespace: "test-namespace",
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`foo`), // pragma: allowlist secret
				},
			},
		},
		{
			name: "secret should be deleted if it exists and dockerConfigJson is empty",
			existingSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tenantImagePullSecretName,
					Namespace: "test-namespace",
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`foo`), // pragma: allowlist secret
				},
			},
			dockerConfigJson: "", // pragma: allowlist secret
			want:             nil,
			expectDelete:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := []runtime.Object{}
			if tt.existingSecret != nil { // pragma: allowlist secret
				objs = append(objs, tt.existingSecret)
			}
			clientSet := fake.NewSimpleClientset(objs...)

			updateCount := 0
			createCount := 0
			deleteCount := 0
			clientSet.PrependReactor("create", "secrets", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
				createCount++
				return false, nil, nil
			})
			clientSet.PrependReactor("update", "secrets", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
				updateCount++
				return false, nil, nil
			})
			clientSet.PrependReactor("delete", "secrets", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
				deleteCount++
				return false, nil, nil
			})

			reconciler := newPullSecretReconciler(clientSet, "test-namespace", []byte(tt.dockerConfigJson))
			ctx := context.Background()
			ctx, err := reconciler.ensurePresent(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)

				if tt.expectUpdate {
					assert.Equal(t, 1, updateCount)
				} else {
					assert.Equal(t, 0, updateCount)
				}

				if tt.expectCreate {
					assert.Equal(t, 1, createCount)
				} else {
					assert.Equal(t, 0, createCount)
				}

				if tt.expectDelete {
					assert.Equal(t, 1, deleteCount)
				} else {
					assert.Equal(t, 0, deleteCount)
				}

				if tt.want != nil {
					got, err := clientSet.CoreV1().Secrets("test-namespace").Get(ctx, tenantImagePullSecretName, metav1.GetOptions{})
					require.NoError(t, err)
					assert.Equal(t, tt.want.Name, got.Name)
					assert.Equal(t, tt.want.Namespace, got.Namespace)
					assert.Equal(t, tt.want.Type, got.Type)
					assert.Equal(t, tt.want.Data, got.Data)
				} else {
					_, err := clientSet.CoreV1().Secrets("test-namespace").Get(ctx, tenantImagePullSecretName, metav1.GetOptions{})
					assert.Error(t, err)
				}
			}
		})
	}

}
