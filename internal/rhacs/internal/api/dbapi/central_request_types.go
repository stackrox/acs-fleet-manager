package dbapi

import (
	"github.com/stackrox/acs-fleet-manager/pkg/api"
)

const (
	CentralRequestListKind = "CentralRequestList"
)

type CentralRequest struct {
	api.Meta

	Status         string `json:"status" gorm:"index"`
	CloudProvider  string `json:"cloud_provider"`
	MultiAZ        bool   `json:"multi_az"`
	Region         string `json:"region"`

	OwnerUser      string `json:"owner_user" gorm:"index"` // TODO: ocm owner?
	OwnerUserAccountId string `json:"owner_account_id"`
	OwnerOrganisation string `json:"owner_organisation" gorm:"index"`
	Name           string `json:"name" gorm:"index"`
	// The DNS host (domain) of the Central service
	Host           string `json:"host"`
	FailedReason   string `json:"failed_reason"`
	// The type of Central instance (eval or standard)
	InstanceType string `json:"instance_type"`
}

type CentralList []*CentralRequest
