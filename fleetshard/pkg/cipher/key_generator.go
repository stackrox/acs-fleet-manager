package cipher

// KeyGenerator defines a Generate method to generate encryption keys
// that can be used for symmetric encryption
type KeyGenerator interface {
	Generate() ([]byte, error)
}
