package environments

import "github.com/stackrox/acs-fleet-manager/pkg/environments"

// NewStageEnvLoader ...
func NewStageEnvLoader() environments.EnvLoader {
	return environments.SimpleEnvLoader{
		"ocm-base-url":                   "https://api.stage.openshift.com",
		"ams-base-url":                   "https://api.stage.openshift.com",
		"enable-ocm-mock":                "false",
		"enable-deny-list":               "true",
		"max-allowed-instances":          "1",
		"enable-central-external-domain": "true",
		"cluster-compute-machine-type":   "m5.2xlarge",
		"admin-authz-config-file":        "config/admin-authz-roles-dev.yaml",
	}
}
