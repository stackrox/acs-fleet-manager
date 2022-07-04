package iam

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

type IAMConfig struct {
	BaseURL                                    string                `json:"base_url"`
	SsoBaseUrl                                 string                `json:"sso_base_url"`
	Debug                                      bool                  `json:"debug"`
	InsecureSkipVerify                         bool                  `json:"insecure-skip-verify"`
	TLSTrustedCertificatesKey                  string                `json:"tls_trusted_certificates_key"`
	TLSTrustedCertificatesValue                string                `json:"tls_trusted_certificates_value"`
	TLSTrustedCertificatesFile                 string                `json:"tls_trusted_certificates_file"`
	OSDClusterIDPRealm                         *IAMRealmConfig       `json:"osd_cluster_idp_realm"`
	RedhatSSORealm                             *IAMRealmConfig       `json:"redhat_sso_config"`
	MaxAllowedServiceAccounts                  int                   `json:"max_allowed_service_accounts"`
	MaxLimitForGetClients                      int                   `json:"max_limit_for_get_clients"`
	ServiceAccounttLimitCheckSkipOrgIdListFile string                `json:"-"`
	ServiceAccounttLimitCheckSkipOrgIdList     []string              `json:"-"`
	AdditionalSSOIssuers                       *AdditionalSSOIssuers `json:"-"`
}

type AdditionalSSOIssuers struct {
	URIs     []string
	JWKSURIs []string
	File     string
	Enabled  bool
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
		AdditionalSSOIssuers:                       &AdditionalSSOIssuers{},
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
	fs.BoolVar(&kc.AdditionalSSOIssuers.Enabled, "enable-additional-sso-issuers", kc.AdditionalSSOIssuers.Enabled, "Enable additional SSO issuer URIs for verifying tokens")
	fs.StringVar(&kc.AdditionalSSOIssuers.File, "additional-sso-issuers-file", kc.AdditionalSSOIssuers.File, "File containing a list of SSO issuer URIs to include for verifying tokens")
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

	// Read the service account limits check skip org ID yaml file
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

	// Read the additional issuers file. This will add additional SSO issuer URIs which shall be used as valid issuers
	// for tokens, i.e. sso.stage.redhat.com.
	if kc.AdditionalSSOIssuers.Enabled {
		err = readAdditionalIssuersFile(kc.AdditionalSSOIssuers.File, kc.AdditionalSSOIssuers)
		if err != nil {
			if os.IsNotExist(err) {
				glog.V(10).Infof("Specified additional SSO issuers file %q does not exist. "+
					"Proceeding as if no additional SSO issuers list was provided", kc.AdditionalSSOIssuers.File)
			} else {
				return err
			}
		}
		if err := kc.AdditionalSSOIssuers.resolveURIs(); err != nil {
			return err
		}
	}

	return nil
}

const (
	openidConfigurationPath = "/.well-known/openid-configuration"
)

type openIdConfiguration struct {
	JwksURI string `json:"jwks_uri"`
}

// setJWKSURIs will set the jwks URIs by taking the issuer URI and fetching the openid-configuration, setting the
// jwks URI dynamically
func (a *AdditionalSSOIssuers) resolveURIs() error {
	client := http.Client{Timeout: time.Minute}
	jwksURIs := make([]string, 0, len(a.URIs))
	for _, issuerURI := range a.URIs {
		cfg, err := getOpenIDConfiguration(client, issuerURI)
		if err != nil {
			return errors.Wrapf(err, "retrieving open-id configuration for %q", issuerURI)
		}
		if cfg.JwksURI == "" {
			return errors.Errorf("no jwks URI found within open-id configuration for %q", issuerURI)
		}
		jwksURIs = append(jwksURIs, cfg.JwksURI)
	}
	a.JWKSURIs = jwksURIs
	return nil
}

func getOpenIDConfiguration(c http.Client, baseURL string) (*openIdConfiguration, error) {
	url := strings.TrimRight(baseURL, "/") + openidConfigurationPath
	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("failed to GET %q, received status code %d", url, resp.StatusCode)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading response")
	}
	var cfg openIdConfiguration
	if err := json.Unmarshal(bytes, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func readAdditionalIssuersFile(file string, endpoints *AdditionalSSOIssuers) error {
	var issuers []string
	if err := shared.ReadYamlFile(file, &issuers); err != nil {
		return err
	}
	endpoints.URIs = issuers
	return nil
}
