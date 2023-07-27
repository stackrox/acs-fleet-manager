package cipher

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
)

type kmsCipher struct {
	keyID string
	kms   *kms.KMS
}

// NewKMSCipher return a new Cipher using AWS KMS with the given keyId
func NewKMSCipher(keyID string) (Cipher, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, fmt.Errorf("unable to create session for KMS client %w", err)
	}

	return kmsCipher{
		keyID: keyID,
		kms:   kms.New(sess),
	}, nil
}

func (k kmsCipher) Encrypt(plaintext []byte) ([]byte, error) {
	encryptInput := &kms.EncryptInput{
		KeyId:     &k.keyID,
		Plaintext: plaintext,
	}

	encryptOut, err := k.kms.Encrypt(encryptInput)
	if err != nil {
		return nil, fmt.Errorf("error encrypting data: %w", err)
	}

	return encryptOut.CiphertextBlob, nil
}

func (k kmsCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	decryptInput := &kms.DecryptInput{
		KeyId:          &k.keyID,
		CiphertextBlob: ciphertext,
	}

	decryptOut, err := k.kms.Decrypt(decryptInput)
	if err != nil {
		return nil, fmt.Errorf("error decrypting data: %w", err)
	}
	return decryptOut.Plaintext, nil
}
