package cipher

import (
	"crypto/rand"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func generateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}

	return key, nil
}

// TestDifferentCipherForSamePlaintext tests the correct usage of nonce
// to ensure encrypting the same plaintext twice does not yield the same cipher text
func TestAES256DifferentCipherForSamePlaintext(t *testing.T) {
	plaintext := []byte("test plaintext")
	key, err := generateKey()
	require.NoError(t, err, "generating key")

	aes, err := NewAES256Cipher(key)
	require.NoError(t, err, "creating cipher")
	cipher1, err := aes.Encrypt(plaintext)
	require.NoError(t, err, "encrypting first plaintext")
	cipher2, err := aes.Encrypt(plaintext)
	require.NoError(t, err, "encrypting second plaintext")

	require.NotEqual(t, cipher1, cipher2, "encrypting same text twice yields same result")
}

func TestAES256EncryptDecryptMatch(t *testing.T) {
	plaintext := []byte("test plaintext")
	key, err := generateKey()
	require.NoError(t, err, "generating key")

	aes, err := NewAES256Cipher(key)
	require.NoError(t, err, "creating cipher")

	cipher, err := aes.Encrypt(plaintext)
	require.NoError(t, err, "encyrpting plaintext")

	decrypted, err := aes.Decrypt(cipher)
	require.NoError(t, err, "decrypting ciphertext")

	require.Equal(t, string(plaintext), string(decrypted), "decrypted string does not match plaintext")
}
