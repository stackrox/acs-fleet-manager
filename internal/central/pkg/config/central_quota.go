package config

import "github.com/stackrox/acs-fleet-manager/pkg/api"

// CentralQuotaConfig ...
type CentralQuotaConfig struct {
	Type                   string `json:"type"`
	AllowEvaluatorInstance bool   `json:"allow_evaluator_instance"`
}

// NewCentralQuotaConfig ...
func NewCentralQuotaConfig() *CentralQuotaConfig {
	return &CentralQuotaConfig{
		Type:                   api.QuotaManagementListQuotaType.String(),
		AllowEvaluatorInstance: true,
	}
}
