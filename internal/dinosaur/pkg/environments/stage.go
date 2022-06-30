package environments

import "github.com/stackrox/acs-fleet-manager/pkg/environments"

func NewStageEnvLoader() environments.EnvLoader {
	return environments.SimpleEnvLoader{
		"ocm-base-url":                         "https://api.stage.openshift.com",
		"ams-base-url":                         "https://api.stage.openshift.com",
		"enable-ocm-mock":                      "false",
		"enable-deny-list":                     "true",
		"max-allowed-instances":                "1",
		"sso-base-url":                         "https://sso.redhat.com",
		"enable-dinosaur-external-certificate": "true",
		"cluster-compute-machine-type":         "m5.2xlarge",
		"enable-additional-sso-issuers":        "true",
		"additional-sso-issuers-file":          "config/additional-sso-issuers.yaml",
		"jwks-file":                            "config/authentication/jwks-file-static.json",
	}
}
