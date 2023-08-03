package cipher

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
)

type kmsCipher struct {
	keyID      string
	kms        *kms.KMS
	dateKeyLen int
}

// NewKMSCipher return a new Cipher using AWS KMS with the given keyId
// The implementation uses the KMS keyID to generate KMS data keys and encrypt data using
// those data keys. This is necessary because encrypting via KMS API caps plaintext length to 4096 bytes
func NewKMSCipher(keyID string) (Cipher, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, fmt.Errorf("unable to create session for KMS client %w", err)
	}

	kmsCipher := kmsCipher{
		keyID: keyID,
		kms:   kms.New(sess),
	}

	dataKeyOut, err := kmsCipher.kms.GenerateDataKey(kmsCipher.dateKeyInput())
	if err != nil {
		return nil, fmt.Errorf("intializing KMS data key length: %w", err)
	}

	kmsCipher.dateKeyLen = len(dataKeyOut.CiphertextBlob)

	return kmsCipher, nil
}

func (k kmsCipher) dateKeyInput() *kms.GenerateDataKeyInput {
	keySpec := kms.DataKeySpecAes256
	return &kms.GenerateDataKeyInput{KeyId: &k.keyID, KeySpec: &keySpec}
}

func (k kmsCipher) Encrypt(plaintext []byte) ([]byte, error) {

	dataKeyOut, err := k.kms.GenerateDataKey(k.dateKeyInput())
	if err != nil {
		return nil, fmt.Errorf("creating KMS data key")
	}

	aesCipher, err := NewAES256Cipher(dataKeyOut.Plaintext)
	if err != nil {
		return nil, fmt.Errorf("creating AES256 cipher from KMS data key %w", err)
	}

	ciphertext, err := aesCipher.Encrypt(plaintext)
	if err != nil {
		return nil, fmt.Errorf("encrypting data: %w", err)
	}

	ciphertext = append(ciphertext, dataKeyOut.CiphertextBlob...)

	return ciphertext, nil
}

func (k kmsCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	keyIndex := len(ciphertext) - k.dateKeyLen
	cipher, encryptedKey := ciphertext[:keyIndex], ciphertext[keyIndex:]

	decryptOut, err := k.kms.Decrypt(&kms.DecryptInput{
		KeyId:          &k.keyID,
		CiphertextBlob: encryptedKey,
	})
	if err != nil {
		return nil, fmt.Errorf("error decrypting data key: %w", err)
	}

	aesCipher, err := NewAES256Cipher(decryptOut.Plaintext)
	if err != nil {
		return nil, fmt.Errorf("creating AES256Cipher from data key: %w", err)
	}

	plaintext, err := aesCipher.Decrypt(cipher)
	if err != nil {
		return nil, fmt.Errorf("decrypting ciphertext: %w", err)
	}

	return plaintext, nil
}
