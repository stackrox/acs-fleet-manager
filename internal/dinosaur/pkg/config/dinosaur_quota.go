package config

import "github.com/stackrox/acs-fleet-manager/pkg/api"

// DinosaurQuotaConfig ...
type DinosaurQuotaConfig struct {
	Type                   string `json:"type"`
	AllowEvaluatorInstance bool   `json:"allow_evaluator_instance"`
}

// NewDinosaurQuotaConfig ...
func NewDinosaurQuotaConfig() *DinosaurQuotaConfig {
	return &DinosaurQuotaConfig{
		Type:                   api.QuotaManagementListQuotaType.String(),
		AllowEvaluatorInstance: true,
	}
}
