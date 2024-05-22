package environments

import "github.com/stackrox/acs-fleet-manager/pkg/environments"

// NewStageEnvLoader ...
func NewStageEnvLoader() environments.EnvLoader {
	return environments.SimpleEnvLoader{
		"ocm-base-url":                        "https://api.stage.openshift.com",
		"ams-base-url":                        "https://api.stage.openshift.com",
		"enable-ocm-mock":                     "false",
		"enable-deny-list":                    "true",
		"max-allowed-instances":               "1",
		"enable-central-external-certificate": "true",
		"cluster-compute-machine-type":        "m5.2xlarge",
		"enable-additional-sso-issuers":       "true",
		"additional-sso-issuers-file":         "config/additional-sso-issuers.yaml",
		"jwks-file":                           "config/jwks-file-static.json",
		"fleetshard-authz-config-file":        "config/fleetshard-authz-development.yaml",
		"admin-authz-config-file":             "config/admin-authz-roles-dev.yaml",
	}
}
