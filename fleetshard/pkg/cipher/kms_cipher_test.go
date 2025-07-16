package cipher

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/stretchr/testify/require"
)

func TestKMSEncryptDecrypt(t *testing.T) {
	if os.Getenv("RUN_AWS_INTEGRATION") != "true" {
		t.Skip("Skip KMS tests. Set RUN_AWS_INTEGRATION=true env variable to enable KMS tests.")
	}

	keyID := os.Getenv("SECRET_ENCRYPTION_KEY_ID")
	require.NotEmpty(t, keyID, "SECRET_ENCRYPTION_KEY_ID not set")

	cipher, err := NewKMSCipher(keyID)
	require.NoError(t, err, "creating KMS cipher")

	plaintext := "This is example plain text"
	plaintextB := []byte(plaintext)
	ciphertextB, err := cipher.Encrypt(plaintextB)
	require.NoError(t, err, "encrypting plaintext")

	decrypted, err := cipher.Decrypt(ciphertextB)
	require.NoError(t, err, "decrypting ciphertext")

	require.NotEqual(t, plaintext, string(ciphertextB))
	require.Equal(t, plaintext, string(decrypted))
}

func TestKMSDataKeyGenerator(t *testing.T) {
	if os.Getenv("RUN_AWS_INTEGRATION") != "true" {
		t.Skip("Skip KMS tests. Set RUN_AWS_INTEGRATION=true env variable to enable KMS tests.")
	}

	keyID := os.Getenv("SECRET_ENCRYPTION_KEY_ID")
	require.NotEmpty(t, keyID, "SECRET_ENCRYPTION_KEY_ID not set")
	generator, err := NewKMSDataKeyGenerator(keyID, types.DataKeySpecAes256)
	require.NoError(t, err)

	key, err := generator.Generate()
	require.NoError(t, err)
	require.Equal(t, 32, len(key))
}
