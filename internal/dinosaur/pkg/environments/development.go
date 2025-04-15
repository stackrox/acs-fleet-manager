// Package environments ...
package environments

import "github.com/stackrox/acs-fleet-manager/pkg/environments"

// NewDevelopmentEnvLoader The development environment is intended for use while developing features, requiring manual verification
func NewDevelopmentEnvLoader() environments.EnvLoader {
	return environments.SimpleEnvLoader{
		"v":                                  "10",
		"ocm-debug":                          "false",
		"ams-base-url":                       "https://api.stage.openshift.com",
		"ocm-base-url":                       "https://api.stage.openshift.com",
		"enable-ocm-mock":                    "true",
		"enable-https":                       "false",
		"enable-metrics-https":               "false",
		"enable-terms-acceptance":            "false",
		"api-server-bindaddress":             "localhost:8000",
		"enable-sentry":                      "false",
		"enable-deny-list":                   "true",
		"enable-instance-limit-control":      "false",
		"enable-central-external-domain":     "false",
		"cluster-compute-machine-type":       "m5.2xlarge",
		"allow-evaluator-instance":           "true",
		"quota-type":                         "quota-management-list",
		"enable-deletion-of-expired-central": "true",
		"dataplane-cluster-scaling-type":     "manual",
		"enable-additional-sso-issuers":      "true",
		"additional-sso-issuers-file":        "config/additional-sso-issuers.yaml",
		"jwks-file":                          "config/jwks-file-static.json",
		"fleetshard-authz-config-file":       "config/fleetshard-authz-development.yaml",
		"central-idp-client-id":              "rhacs-ms-dev",
		"central-idp-issuer":                 "https://sso.stage.redhat.com/auth/realms/redhat-external",
		"admin-authz-config-file":            "config/admin-authz-roles-dev.yaml",
		"enable-leader-election":             "false",
		"kubernetes-issuer-enabled":          "true",
		"kubernetes-issuer-uri":              "https://127.0.0.1:6443",
	}
}
