package cipher

import (
	"context"
	"fmt"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

type kmsCipher struct {
	keyID      string
	kms        *kms.Client
	dateKeyLen int
}

// NewKMSCipher return a new Cipher using AWS KMS with the given keyId
// The implementation uses the KMS keyID to generate KMS data keys and encrypt data using
// those data keys. This is necessary because encrypting via KMS API caps plaintext length to 4096 bytes
func NewKMSCipher(keyID string) (Cipher, error) {
	cfg, err := awsConfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config for KMS client %w", err)
	}

	kmsCipher := kmsCipher{
		keyID: keyID,
		kms:   kms.NewFromConfig(cfg),
	}

	dataKeyOut, err := kmsCipher.kms.GenerateDataKey(context.TODO(), kmsCipher.dateKeyInput())
	if err != nil {
		return nil, fmt.Errorf("intializing KMS data key length: %w", err)
	}

	kmsCipher.dateKeyLen = len(dataKeyOut.CiphertextBlob)

	return kmsCipher, nil
}

func (k kmsCipher) dateKeyInput() *kms.GenerateDataKeyInput {
	keySpec := types.DataKeySpecAes256
	return &kms.GenerateDataKeyInput{KeyId: &k.keyID, KeySpec: keySpec}
}

func (k kmsCipher) Encrypt(plaintext []byte) ([]byte, error) {

	dataKeyOut, err := k.kms.GenerateDataKey(context.TODO(), k.dateKeyInput())
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

	decryptOut, err := k.kms.Decrypt(context.TODO(), &kms.DecryptInput{
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

// KMSDataKeyGenerator implements KeyGenerator using AWS KMS
type KMSDataKeyGenerator struct {
	keyID          string
	kms            *kms.Client
	kmsDataKeySpec types.DataKeySpec
}

// NewKMSDataKeyGenerator initiates a AWS KMS session and
// returns a new instance of KMSDataKeyGenerator
func NewKMSDataKeyGenerator(keyID string, keySpec types.DataKeySpec) (*KMSDataKeyGenerator, error) {
	cfg, err := awsConfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config for KMS client %w", err)
	}

	return &KMSDataKeyGenerator{
		keyID:          keyID,
		kms:            kms.NewFromConfig(cfg),
		kmsDataKeySpec: keySpec,
	}, nil
}

// Generate generates a KMS data key with the KMSDataKeyGenerator configuration
func (g KMSDataKeyGenerator) Generate() ([]byte, error) {
	dateKeyOut, err := g.kms.GenerateDataKey(context.TODO(), &kms.GenerateDataKeyInput{KeyId: &g.keyID, KeySpec: g.kmsDataKeySpec})
	if err != nil {
		return nil, fmt.Errorf("generating kms data key: %w", err)
	}

	return dateKeyOut.Plaintext, nil
}
