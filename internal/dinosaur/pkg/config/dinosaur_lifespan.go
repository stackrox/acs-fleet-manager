package config

// DinosaurLifespanConfig ...
type DinosaurLifespanConfig struct {
	EnableDeletionOfExpiredDinosaur bool
	DinosaurLifespanInHours         int
}

// NewDinosaurLifespanConfig ...
func NewDinosaurLifespanConfig() *DinosaurLifespanConfig {
	return &DinosaurLifespanConfig{
		EnableDeletionOfExpiredDinosaur: true,
		DinosaurLifespanInHours:         48,
	}
}
