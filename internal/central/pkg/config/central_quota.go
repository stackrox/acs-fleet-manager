package config

import "github.com/stackrox/acs-fleet-manager/pkg/api"

// CentralQuotaConfig ...
type CentralQuotaConfig struct {
	Type                   string `json:"type"`
	AllowEvaluatorInstance bool   `json:"allow_evaluator_instance"`
	// InternalCentralIDs is a list of Central IDs that should be ignored for quota checks
	InternalCentralIDs []string `json:"internal_central_ids"`
}

// NewCentralQuotaConfig ...
func NewCentralQuotaConfig() *CentralQuotaConfig {
	return &CentralQuotaConfig{
		Type:                   api.QuotaManagementListQuotaType.String(),
		AllowEvaluatorInstance: true,
		InternalCentralIDs:     []string{},
	}
}
