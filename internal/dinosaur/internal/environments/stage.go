package environments

import "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"

func NewStageEnvLoader() environments.EnvLoader {
	return environments.SimpleEnvLoader{
		"ocm-base-url":                         "https://api.stage.openshift.com",
		"ams-base-url":                         "https://api.stage.openshift.com",
		"enable-ocm-mock":                      "false",
		"enable-deny-list":                     "true",
		"max-allowed-instances":                "1",
		"sso-base-url":                         "https://identity.api.stage.openshift.com",
		"sso-realm":                            "rhoas",
		"enable-dinosaur-external-certificate": "true",
		"cluster-compute-machine-type":         "m5.2xlarge",
	}
}
