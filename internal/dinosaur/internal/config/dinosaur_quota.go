package config

import "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"

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
