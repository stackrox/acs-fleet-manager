package cipher

import (
	"encoding/base64"
	"fmt"
)

// LocalBase64Cipher simulates encryption by using base64 encoding and decoding
// Warning: Only use this for development it does not encrypt data
type LocalBase64Cipher struct {
}

// NewLocalBase64Cipher returns a new Cipher using the given key
func NewLocalBase64Cipher() (Cipher, error) {
	return LocalBase64Cipher{}, nil
}

var _ Cipher = LocalBase64Cipher{}

// Encrypt implementes the logic to encode plaintext with base64
func (a LocalBase64Cipher) Encrypt(plaintext []byte) ([]byte, error) {
	enc := base64.StdEncoding.EncodeToString(plaintext)
	return []byte(enc), nil
}

// Decrypt implements the logic to decode base64 text to plaintext
func (a LocalBase64Cipher) Decrypt(ciphertext []byte) ([]byte, error) {
	plaintext, err := base64.StdEncoding.DecodeString(string(ciphertext))
	if err != nil {
		return nil, fmt.Errorf("decoding base64 string %w", err)
	}
	return plaintext, nil
}
