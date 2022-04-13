package config

type DinosaurLifespanConfig struct {
	EnableDeletionOfExpiredDinosaur bool
	DinosaurLifespanInHours         int
}

func NewDinosaurLifespanConfig() *DinosaurLifespanConfig {
	return &DinosaurLifespanConfig{
		EnableDeletionOfExpiredDinosaur: true,
		DinosaurLifespanInHours:         48,
	}
}
