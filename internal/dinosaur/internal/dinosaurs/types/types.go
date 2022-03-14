package types

import "github.com/stackrox/acs-fleet-manager/pkg/client/ocm"

type DinosaurInstanceType string

const (
	EVAL     DinosaurInstanceType = "eval"
	STANDARD DinosaurInstanceType = "standard"
)

func (t DinosaurInstanceType) String() string {
	return string(t)
}

func (t DinosaurInstanceType) GetQuotaType() ocm.DinosaurQuotaType {
	if t == STANDARD {
		return ocm.StandardQuota
	}
	return ocm.EvalQuota
}
