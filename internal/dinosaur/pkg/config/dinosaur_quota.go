package config

import "github.com/stackrox/acs-fleet-manager/pkg/api"

type DinosaurQuotaConfig struct {
	Type                   string `json:"type"`
	AllowEvaluatorInstance bool   `json:"allow_evaluator_instance"`
}

func NewDinosaurQuotaConfig() *DinosaurQuotaConfig {
	return &DinosaurQuotaConfig{
		Type:                   api.QuotaManagementListQuotaType.String(),
		AllowEvaluatorInstance: true,
	}
}
