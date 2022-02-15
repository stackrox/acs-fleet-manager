package environments

import "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"

func NewProductionEnvLoader() environments.EnvLoader {
	return environments.SimpleEnvLoader{
		"v":                                    "1",
		"ocm-debug":                            "false",
		"enable-ocm-mock":                      "false",
		"enable-sentry":                        "true",
		"enable-deny-list":                     "true",
		"max-allowed-instances":                "1",
		"sso-realm":                            "rhoas",
		"sso-base-url":                         "https://identity.api.openshift.com",
		"enable-dinosaur-external-certificate": "true",
		"cluster-compute-machine-type":         "m5.2xlarge",
	}
}
