// Package environments ...
package environments

import "github.com/stackrox/acs-fleet-manager/pkg/environments"

// NewDevelopmentEnvLoader The development environment is intended for use while developing features, requiring manual verification
func NewDevelopmentEnvLoader() environments.EnvLoader {
	return environments.SimpleEnvLoader{
		"v":                                               "10",
		"ocm-debug":                                       "false",
		"ams-base-url":                                    "https://api.stage.openshift.com",
		"ocm-base-url":                                    "https://api.stage.openshift.com",
		"enable-ocm-mock":                                 "false",
		"enable-https":                                    "false",
		"enable-metrics-https":                            "false",
		"enable-terms-acceptance":                         "false",
		"api-server-bindaddress":                          "localhost:8000",
		"enable-sentry":                                   "false",
		"enable-deny-list":                                "true",
		"enable-instance-limit-control":                   "false",
		"sso-base-url":                                    "https://sso.redhat.com",
		"enable-central-external-certificate":             "false",
		"cluster-compute-machine-type":                    "m5.2xlarge",
		"allow-evaluator-instance":                        "true",
		"quota-type":                                      "quota-management-list",
		"enable-deletion-of-expired-central":              "true",
		"dataplane-cluster-scaling-type":                  "manual",
		"central-operator-addon-id":                       "managed-central-qe",
		"fleetshard-addon-id":                             "fleetshard-operator-qe",
		"observability-red-hat-sso-auth-server-url":       "https://sso.redhat.com/auth",
		"observability-red-hat-sso-realm":                 "redhat-external",
		"observability-red-hat-sso-token-refresher-url":   "http://localhost:8085",
		"observability-red-hat-sso-observatorium-gateway": "https://observatorium-mst.api.stage.openshift.com",
		"observability-red-hat-sso-tenant":                "manageddinosaur",
		"enable-additional-sso-issuers":                   "true",
		"additional-sso-issuers-file":                     "config/additional-sso-issuers.yaml",
		"jwks-file":                                       "config/jwks-file-static.json",
		"fleetshard-authz-config-file":                    "config/fleetshard-authz-org-ids-development.yaml",
		"central-idp-client-id":                           "rhacs-ms-dev",
		"central-idp-issuer":                              "https://sso.stage.redhat.com/auth/realms/redhat-external",
		"admin-authz-config-file":                         "config/admin-authz-roles-dev.yaml",
	}
}
