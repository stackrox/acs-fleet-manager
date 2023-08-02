// Package cipher defines encryption and decryption methods used by fleetshard-sync
package cipher

import (
	"fmt"

	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
)

//go:generate moq -out cipher_moq.go . Cipher

// Cipher is the interface used to encrypt and decrypt content
type Cipher interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

// NewCipher returns a new object implementing cipher, based on the Type defined in config
func NewCipher(config *config.Config) (Cipher, error) {
	encryptionType := config.SecretEncryption.Type

	if encryptionType == "local" {
		return NewLocalBase64Cipher()
	}

	if encryptionType == "kms" {
		return NewKMSCipher(config.SecretEncryption.KeyID)
	}

	return nil, fmt.Errorf("no Cipher implementation for SecretEncryption.Type: %s", encryptionType)
}
