package config

import "github.com/stackrox/acs-fleet-manager/pkg/api"

// CentralQuotaConfig ...
type CentralQuotaConfig struct {
	Type                   string `json:"type"`
	AllowEvaluatorInstance bool   `json:"allow_evaluator_instance"`
	// InternalOrganisationIds is a list of organisation IDs that shhould be ignored for quota checks
	InternalOrganisationIDs []string `json:"internal_organisation_ids"`
}

// NewCentralQuotaConfig ...
func NewCentralQuotaConfig() *CentralQuotaConfig {
	return &CentralQuotaConfig{
		Type:                    api.QuotaManagementListQuotaType.String(),
		AllowEvaluatorInstance:  true,
		InternalOrganisationIDs: []string{},
	}
}
