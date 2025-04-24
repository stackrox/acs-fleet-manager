// Package types ...
package types

import (
	ocm "github.com/stackrox/acs-fleet-manager/pkg/client/ocm/impl"
)

// CentralInstanceType ...
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
