package environments

import "github.com/stackrox/acs-fleet-manager/pkg/environments"

// NewProductionEnvLoader ...
func NewProductionEnvLoader() environments.EnvLoader {
	return environments.SimpleEnvLoader{
		"v":                                    "1",
		"ocm-debug":                            "false",
		"enable-ocm-mock":                      "false",
		"enable-sentry":                        "true",
		"enable-deny-list":                     "true",
		"max-allowed-instances":                "1",
		"sso-base-url":                         "https://sso.redhat.com",
		"enable-dinosaur-external-certificate": "true",
		"cluster-compute-machine-type":         "m5.2xlarge",
	}
}
