package iam

import (
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"os"

	"github.com/spf13/pflag"
)

type IAMConfig struct {
	BaseURL                                    string          `json:"base_url"`
	SsoBaseUrl                                 string          `json:"sso_base_url"`
	Debug                                      bool            `json:"debug"`
	InsecureSkipVerify                         bool            `json:"insecure-skip-verify"`
	TLSTrustedCertificatesKey                  string          `json:"tls_trusted_certificates_key"`
	TLSTrustedCertificatesValue                string          `json:"tls_trusted_certificates_value"`
	TLSTrustedCertificatesFile                 string          `json:"tls_trusted_certificates_file"`
	OSDClusterIDPRealm                         *IAMRealmConfig `json:"osd_cluster_idp_realm"`
	RedhatSSORealm                             *IAMRealmConfig `json:"redhat_sso_config"`
	MaxAllowedServiceAccounts                  int             `json:"max_allowed_service_accounts"`
	MaxLimitForGetClients                      int             `json:"max_limit_for_get_clients"`
	ServiceAccounttLimitCheckSkipOrgIdListFile string          `json:"-"`
	ServiceAccounttLimitCheckSkipOrgIdList     []string        `json:"-"`
}

type IAMRealmConfig struct {
	BaseURL          string `json:"base_url"`
	Realm            string `json:"realm"`
	ClientID         string `json:"client-id"`
	ClientIDFile     string `json:"client-id_file"`
	ClientSecret     string `json:"client-secret"`
	ClientSecretFile string `json:"client-secret_file"`
	GrantType        string `json:"grant_type"`
	TokenEndpointURI string `json:"token_endpoint_uri"`
	JwksEndpointURI  string `json:"jwks_endpoint_uri"`
	ValidIssuerURI   string `json:"valid_issuer_uri"`
	APIEndpointURI   string `json:"api_endpoint_uri"`
}

func (c *IAMRealmConfig) setDefaultURIs(baseURL string) {
	c.BaseURL = baseURL
	c.ValidIssuerURI = baseURL + "/auth/realms/" + c.Realm
	c.JwksEndpointURI = baseURL + "/auth/realms/" + c.Realm + "/protocol/openid-connect/certs"
	c.TokenEndpointURI = baseURL + "/auth/realms/" + c.Realm + "/protocol/openid-connect/token"
}

func NewKeycloakConfig() *IAMConfig {
	kc := &IAMConfig{
		SsoBaseUrl: "https://sso.redhat.com",
		OSDClusterIDPRealm: &IAMRealmConfig{
			ClientIDFile:     "secrets/osd-idp-keycloak-service.clientId",
			ClientSecretFile: "secrets/osd-idp-keycloak-service.clientSecret",
			GrantType:        "client_credentials",
		},
		Debug:                 false,
		InsecureSkipVerify:    false,
		MaxLimitForGetClients: 100,
		RedhatSSORealm: &IAMRealmConfig{
			APIEndpointURI:   "/auth/realms/redhat-external",
			Realm:            "redhat-external",
			ClientIDFile:     "secrets/redhatsso-service.clientId",
			ClientSecretFile: "secrets/redhatsso-service.clientSecret",
			GrantType:        "client_credentials",
		},
		TLSTrustedCertificatesFile:                 "secrets/keycloak-service.crt",
		TLSTrustedCertificatesKey:                  "keycloak.crt",
		MaxAllowedServiceAccounts:                  50,
		ServiceAccounttLimitCheckSkipOrgIdListFile: "config/service-account-limits-check-skip-org-id-list.yaml",
	}
	return kc
}

func (kc *IAMConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&kc.BaseURL, "sso-base-url", kc.BaseURL, "The base URL of the sso, integration by default")
	fs.StringVar(&kc.TLSTrustedCertificatesFile, "osd-sso-cert-file", kc.TLSTrustedCertificatesFile, "File containing tls cert for the osd-sso. Useful when osd-sso uses a self-signed certificate. If the provided file does not exist, is the empty string or the provided file content is empty then no custom OSD SSO certificate is used")
	fs.BoolVar(&kc.Debug, "sso-debug", kc.Debug, "Debug flag for Keycloak API")
	fs.BoolVar(&kc.InsecureSkipVerify, "sso-insecure", kc.InsecureSkipVerify, "Disable tls verification with sso")
	fs.StringVar(&kc.OSDClusterIDPRealm.ClientIDFile, "osd-idp-sso-client-id-file", kc.OSDClusterIDPRealm.ClientIDFile, "File containing Keycloak privileged account client-id that has access to the OSD Cluster IDP realm")
	fs.StringVar(&kc.OSDClusterIDPRealm.ClientSecretFile, "osd-idp-sso-client-secret-file", kc.OSDClusterIDPRealm.ClientSecretFile, "File containing Keycloak privileged account client-secret that has access to the OSD Cluster IDP realm")
	fs.StringVar(&kc.OSDClusterIDPRealm.Realm, "osd-idp-sso-realm", kc.OSDClusterIDPRealm.Realm, "Realm for OSD cluster IDP clients in the sso")
	fs.IntVar(&kc.MaxAllowedServiceAccounts, "max-allowed-service-accounts", kc.MaxAllowedServiceAccounts, "Max allowed service accounts per org")
	fs.IntVar(&kc.MaxLimitForGetClients, "max-limit-for-sso-get-clients", kc.MaxLimitForGetClients, "Max limits for SSO get clients")
	fs.StringVar(&kc.RedhatSSORealm.ClientIDFile, "redhat-sso-client-id-file", kc.RedhatSSORealm.ClientIDFile, "File containing Keycloak privileged account client-id that has access to the OSD Cluster IDP realm")
	fs.StringVar(&kc.RedhatSSORealm.ClientSecretFile, "redhat-sso-client-secret-file", kc.RedhatSSORealm.ClientSecretFile, "File containing Keycloak privileged account client-secret that has access to the OSD Cluster IDP realm")
	fs.StringVar(&kc.SsoBaseUrl, "redhat-sso-base-url", kc.SsoBaseUrl, "The base URL of the SSO, integration by default")
	fs.StringVar(&kc.ServiceAccounttLimitCheckSkipOrgIdListFile, "service-account-limits-check-skip-org-id-list-file", kc.ServiceAccounttLimitCheckSkipOrgIdListFile, "File containing a list of Org IDs for which service account limits check will be skipped")
}

func (kc *IAMConfig) ReadFiles() error {
	err := shared.ReadFileValueString(kc.OSDClusterIDPRealm.ClientIDFile, &kc.OSDClusterIDPRealm.ClientID)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(kc.OSDClusterIDPRealm.ClientSecretFile, &kc.OSDClusterIDPRealm.ClientSecret)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(kc.OSDClusterIDPRealm.ClientSecretFile, &kc.OSDClusterIDPRealm.ClientSecret)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(kc.RedhatSSORealm.ClientIDFile, &kc.RedhatSSORealm.ClientID)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(kc.RedhatSSORealm.ClientSecretFile, &kc.RedhatSSORealm.ClientSecret)
	if err != nil {
		return err
	}

	// We read the OSD SSO TLS certificate file. If it does not exist we
	// intentionally continue as if it was not provided
	err = shared.ReadFileValueString(kc.TLSTrustedCertificatesFile, &kc.TLSTrustedCertificatesValue)
	if err != nil {
		if os.IsNotExist(err) {
			glog.V(10).Infof("Specified OSD SSO TLS certificate file %q does not exist. Proceeding as if OSD SSO TLS certificate was not provided", kc.TLSTrustedCertificatesFile)
		} else {
			return err
		}
	}

	//Read the service account limits check skip org ID yaml file
	err = shared.ReadYamlFile(kc.ServiceAccounttLimitCheckSkipOrgIdListFile, &kc.ServiceAccounttLimitCheckSkipOrgIdList)
	if err != nil {
		if os.IsNotExist(err) {
			glog.V(10).Infof("Specified service account limits skip org IDs  file %q does not exist. Proceeding as if no service account org ID skip list was provided", kc.ServiceAccounttLimitCheckSkipOrgIdListFile)
		} else {
			return err
		}
	}

	kc.OSDClusterIDPRealm.setDefaultURIs(kc.BaseURL)
	kc.RedhatSSORealm.setDefaultURIs(kc.SsoBaseUrl)

	return nil
}
