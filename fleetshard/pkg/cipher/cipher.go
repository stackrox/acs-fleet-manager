// Package cipher defines encryption and decryption methods used by fleetshard-sync
package cipher

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

const keySize = 32

//go:generate moq -out cipher_moq.go . Cipher

// Cipher is the interface used to encrypt and decrypt content
type Cipher interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

// AES256Cipher implements encryption and decryption using AES256 GCM
type AES256Cipher struct {
	aesgcm cipher.AEAD
}

// NewAES256Cipher returns a new Cipher using the given key
func NewAES256Cipher(key []byte) (Cipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("creating AES256Cipher, key does not match required lenght of %d", keySize)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher block: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating AES GCM cipher %w", err)
	}

	return AES256Cipher{aesgcm: aesgcm}, nil
}

var _ Cipher = AES256Cipher{}

// Encrypt implementes the logic to encrypt plaintext
func (a AES256Cipher) Encrypt(plaintext []byte) ([]byte, error) {

	nonce := make([]byte, a.aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generating nonce for encryption %w", err)
	}

	ciphertext := a.aesgcm.Seal(nil, nonce, plaintext, nil)

	// append nonce to ciphertext so decrypt can use it
	ciphertext = append(ciphertext, nonce...)

	return ciphertext, nil
}

// Decrypt implements the logic to decrypt ciphertext, it assumes
// a nonce has been apended to ciphertext at encryption
func (a AES256Cipher) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceIndex := len(ciphertext) - a.aesgcm.NonceSize()
	cipher, nonce := ciphertext[:nonceIndex], ciphertext[nonceIndex:]

	plaintext, err := a.aesgcm.Open(nil, nonce, cipher, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting ciphertext: %w", err)
	}

	return plaintext, nil
}
