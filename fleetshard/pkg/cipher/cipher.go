// Package cipher defines encryption and decryption methods used by fleetshard-sync
package cipher

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/kms/types"
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

// NewKeyGenerator return a new object implementing KeyGenerator based on the type defined in config
func NewKeyGenerator(config *config.Config) (KeyGenerator, error) {
	encryptionType := config.SecretEncryption.Type

	if encryptionType == "local" {
		return AES256KeyGenerator{}, nil
	}

	if encryptionType == "kms" {
		return NewKMSDataKeyGenerator(config.SecretEncryption.KeyID, types.DataKeySpecAes256)
	}

	return nil, fmt.Errorf("no KeyGenerator implementation for SecretEncryption.Type: %s", encryptionType)
}
