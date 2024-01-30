// Package types ...
package types

import (
	ocm "github.com/stackrox/acs-fleet-manager/pkg/client/ocm/impl"
)

// DinosaurInstanceType ...
type DinosaurInstanceType string

// EVAL ...
const (
	EVAL     DinosaurInstanceType = "eval"
	STANDARD DinosaurInstanceType = "standard"
)

// String ...
func (t DinosaurInstanceType) String() string {
	return string(t)
}

// GetQuotaType ...
func (t DinosaurInstanceType) GetQuotaType() ocm.DinosaurQuotaType {
	if t == STANDARD {
		return ocm.StandardQuota
	}
	return ocm.EvalQuota
}
