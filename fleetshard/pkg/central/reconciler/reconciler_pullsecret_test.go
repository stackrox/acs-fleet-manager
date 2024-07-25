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
			objs := []ctrlClient.Object{}
			if tt.existingSecret != nil { // pragma: allowlist secret
				objs = append(objs, tt.existingSecret)
			}

			updateCount := 0
			createCount := 0
			deleteCount := 0

			client := fake2.NewClientBuilder().
				WithInterceptorFuncs(interceptor.Funcs{
					Create: func(ctx context.Context, client ctrlClient.WithWatch, obj ctrlClient.Object, opts ...ctrlClient.CreateOption) error {
						createCount++
						return client.Create(ctx, obj, opts...)
					},
					Update: func(ctx context.Context, client ctrlClient.WithWatch, obj ctrlClient.Object, opts ...ctrlClient.UpdateOption) error {
						updateCount++
						return client.Update(ctx, obj, opts...)
					},
					Delete: func(ctx context.Context, client ctrlClient.WithWatch, obj ctrlClient.Object, opts ...ctrlClient.DeleteOption) error {
						deleteCount++
						return client.Delete(ctx, obj, opts...)
					},
				}).
				WithObjects(objs...).
				Build()

			reconciler := newPullSecretReconciler(client, "test-namespace", []byte(tt.dockerConfigJson))
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

				objectKey := ctrlClient.ObjectKey{Namespace: "test-namespace", Name: tenantImagePullSecretName}

				if tt.want != nil {
					var got corev1.Secret
					err := client.Get(ctx, objectKey, &got)
					require.NoError(t, err)
					assert.Equal(t, tt.want.Name, got.Name)
					assert.Equal(t, tt.want.Namespace, got.Namespace)
					assert.Equal(t, tt.want.Type, got.Type)
					assert.Equal(t, tt.want.Data, got.Data)
				} else {
					err := client.Get(ctx, objectKey, &corev1.Secret{})
					assert.Error(t, err)
				}
			}
		})
	}

}
