package types

import "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/ocm"

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
