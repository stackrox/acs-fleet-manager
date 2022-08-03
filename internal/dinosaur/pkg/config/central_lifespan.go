package config

// CentralLifespanConfig ...
type CentralLifespanConfig struct {
	EnableDeletionOfExpiredCentral bool
	CentralLifespanInHours         int
}

// NewCentralLifespanConfig ...
func NewCentralLifespanConfig() *CentralLifespanConfig {
	return &CentralLifespanConfig{
		EnableDeletionOfExpiredCentral: true,
		CentralLifespanInHours:         48,
	}
}
