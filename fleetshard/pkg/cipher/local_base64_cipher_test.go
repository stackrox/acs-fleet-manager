package cipher

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBase64EncryptDecryptMatch(t *testing.T) {
	plaintext := []byte("test plaintext")

	b64Cipher, err := NewLocalBase64Cipher()
	require.NoError(t, err, "creating cipher")

	cipher, err := b64Cipher.Encrypt(plaintext)
	require.NoError(t, err, "encyrpting plaintext")

	decrypted, err := b64Cipher.Decrypt(cipher)
	require.NoError(t, err, "decrypting ciphertext")

	require.Equal(t, string(plaintext), string(decrypted), "decrypted string does not match plaintext")
}
