package reconciler

import (
	"context"
	"encoding/base64"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/cipher"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	fake2 "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

type getCentralFn func(ctx context.Context, centralID string) (private.ManagedCentral, *http.Response, error)
type mockCentralGetter struct {
	getCentralFn getCentralFn
}

func (m mockCentralGetter) GetCentral(ctx context.Context, centralID string) (private.ManagedCentral, *http.Response, error) {
	return m.getCentralFn(ctx, centralID)
}

func Test_secretRestoreReconciler(t *testing.T) {
	testCases := []struct {
		name                     string
		buildCentral             func() private.ManagedCentral
		mockObjects              []runtime.Object
		getCentralFn             getCentralFn
		expectedErrorMsgContains string
		expectedObjects          []runtime.Object
	}{
		{
			name: "no error for SecretsStored not set",
			buildCentral: func() private.ManagedCentral {
				return simpleManagedCentral
			},
		},
		{
			name: "no error for existing secrets in SecretsStored",
			buildCentral: func() private.ManagedCentral {
				newCentral := simpleManagedCentral
				newCentral.Metadata.SecretsStored = []string{"central-tls", "central-db-password"}
				return newCentral
			},
			mockObjects: []runtime.Object{
				centralTLSSecretObject(),
				centralDBPasswordSecretObject(),
			},
		},
		{
			name: "return errors from fleetmanager",
			buildCentral: func() private.ManagedCentral {
				newCentral := simpleManagedCentral
				newCentral.Metadata.SecretsStored = []string{"central-tls", "central-db-password"}
				return newCentral
			},
			mockObjects: []runtime.Object{
				centralTLSSecretObject(),
			},
			getCentralFn: func(ctx context.Context, centralID string) (private.ManagedCentral, *http.Response, error) {
				return private.ManagedCentral{}, nil, errors.New("test error")
			},
			expectedErrorMsgContains: "failed to load secrets for central cb45idheg5ip6dq1jo4g: test error",
		},
		{
			// force encrypt error by using non base64 value for central-db-password
			name: "return errors from decryptSecrets",
			buildCentral: func() private.ManagedCentral {
				newCentral := simpleManagedCentral
				newCentral.Metadata.SecretsStored = []string{"central-tls", "central-db-password"}
				return newCentral
			},
			mockObjects: []runtime.Object{
				centralTLSSecretObject(),
			},
			getCentralFn: func(ctx context.Context, centralID string) (private.ManagedCentral, *http.Response, error) {
				returnCentral := simpleManagedCentral
				returnCentral.Metadata.Secrets = map[string]string{"central-db-password": "testpw"}
				return returnCentral, nil, nil
			},
			expectedErrorMsgContains: "failed to decrypt secrets for central",
		},
		{
			name: "expect secrets to exist after secret restore",
			buildCentral: func() private.ManagedCentral {
				newCentral := simpleManagedCentral
				newCentral.Metadata.SecretsStored = []string{"central-tls", "central-db-password"}
				return newCentral
			},
			getCentralFn: func(ctx context.Context, centralID string) (private.ManagedCentral, *http.Response, error) {
				returnCentral := simpleManagedCentral
				centralTLS := `{"metadata":{"name":"central-tls","namespace":"rhacs-cb45idheg5ip6dq1jo4g","creationTimestamp":null}}`
				centralDBPW := `{"metadata":{"name":"central-db-password","namespace":"rhacs-cb45idheg5ip6dq1jo4g","creationTimestamp":null}}`

				encode := base64.StdEncoding.EncodeToString
				// we need to encode twice, once for b64 test cipher used
				// once for the b64 encoding done to transfer secret data via API
				returnCentral.Metadata.Secrets = map[string]string{
					"central-tls":         encode([]byte(encode([]byte(centralTLS)))),
					"central-db-password": encode([]byte(encode([]byte(centralDBPW)))),
				}
				return returnCentral, nil, nil
			},
			expectedObjects: []runtime.Object{
				centralTLSSecretObject(),
				centralDBPasswordSecretObject(),
			},
		},
	}

	testCipher, err := cipher.NewLocalBase64Cipher()
	require.NoError(t, err)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			ctx := context.Background()
			ctx = withManagedCentral(ctx, tc.buildCentral())

			fakeClient := fake2.NewFakeClient(tc.mockObjects...)
			centralGetter := mockCentralGetter{getCentralFn: tc.getCentralFn}

			r := newSecretRestoreReconciler(fakeClient, centralGetter, testCipher)

			_, err = r.ensurePresent(ctx)

			if err != nil && tc.expectedErrorMsgContains != "" {
				require.Contains(t, err.Error(), tc.expectedErrorMsgContains)
			} else {
				require.NoError(t, err)
			}

			for _, obj := range tc.expectedObjects {
				wantObj, ok := obj.(*corev1.Secret)
				require.True(t, ok, "expected object is not a Secret")
				objectKey := ctrlClient.ObjectKey{Namespace: wantObj.GetNamespace(), Name: wantObj.GetName()}
				err := fakeClient.Get(context.Background(), objectKey, &corev1.Secret{})
				require.NoErrorf(t, err, "finding expected object %s/%s", wantObj.GetNamespace(), wantObj.GetName())
			}

		})
	}
}
