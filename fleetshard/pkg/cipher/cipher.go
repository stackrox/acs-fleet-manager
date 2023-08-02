// Package cipher defines encryption and decryption methods used by fleetshard-sync
package cipher

//go:generate moq -out cipher_moq.go . Cipher

// Cipher is the interface used to encrypt and decrypt content
type Cipher interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}
