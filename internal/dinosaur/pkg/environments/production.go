package environments

import "github.com/stackrox/acs-fleet-manager/pkg/environments"

// NewProductionEnvLoader ...
func NewProductionEnvLoader() environments.EnvLoader {
	return environments.SimpleEnvLoader{
		"ocm-base-url":                        "https://api.openshift.com",
		"ams-base-url":                        "https://api.openshift.com",
		"v":                                   "1",
		"ocm-debug":                           "false",
		"enable-ocm-mock":                     "false",
		"enable-sentry":                       "true",
		"enable-deny-list":                    "true",
		"max-allowed-instances":               "1",
		"enable-central-external-certificate": "true",
		"cluster-compute-machine-type":        "m5.2xlarge",
	}
}
