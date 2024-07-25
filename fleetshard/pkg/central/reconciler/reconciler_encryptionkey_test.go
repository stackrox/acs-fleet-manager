package reconciler

import (
	"context"
	"testing"

	"encoding/base64"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/cipher"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	centralNotifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fake2 "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_encryptionKeyReconciler(t *testing.T) {

	fakeClient := fake2.NewFakeClient()
	r := newEncryptionKeyReconciler(fakeClient, cipher.AES256KeyGenerator{})
	central := private.ManagedCentral{
		Metadata: private.ManagedCentralAllOfMetadata{
			Namespace: "test-namespace",
		},
	}
	ctx := withManagedCentral(context.Background(), central)

	_, err := r.ensurePresent(ctx)
	require.NoError(t, err)

	var centralEncryptionSecret corev1.Secret
	key := client.ObjectKey{Namespace: "test-namespace", Name: centralEncryptionKeySecretName}
	err = fakeClient.Get(ctx, key, &centralEncryptionSecret)
	require.NoError(t, err)
	require.Contains(t, centralEncryptionSecret.Data, "key-chain.yaml")

	var keyChain centralNotifierUtils.KeyChain
	err = yaml.Unmarshal(centralEncryptionSecret.Data["key-chain.yaml"], &keyChain) // pragma: allowlist secret
	require.NoError(t, err)
	require.Equal(t, 0, keyChain.ActiveKeyIndex)
	require.Equal(t, 1, len(keyChain.KeyMap))

	encKey, err := base64.StdEncoding.DecodeString(keyChain.KeyMap[keyChain.ActiveKeyIndex])
	require.NoError(t, err)
	expectedKeyLen := 32 // 256 bits key
	require.Equal(t, expectedKeyLen, len(encKey))
}
