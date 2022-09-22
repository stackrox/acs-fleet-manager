package types

import "github.com/stackrox/acs-fleet-manager/pkg/client/ocm"

// CentralInstanceType represents the type of instance for central. It can either be EVAL or STANDARD.
type CentralInstanceType string

// EVAL ...
const (
	EVAL     CentralInstanceType = "eval"
	STANDARD CentralInstanceType = "standard"
)

// String ...
func (t CentralInstanceType) String() string {
	return string(t)
}

// GetQuotaType ...
func (t CentralInstanceType) GetQuotaType() ocm.CentralQuotaType {
	if t == STANDARD {
		return ocm.StandardQuota
	}
	return ocm.EvalQuota
}
