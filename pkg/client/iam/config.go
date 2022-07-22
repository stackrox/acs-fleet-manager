package iam

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"

	"github.com/spf13/pflag"
)

// IAMConfig ...
type IAMConfig struct {
	BaseURL                                    string                `json:"base_url"`
	SsoBaseURL                                 string                `json:"sso_base_url"`
	Debug                                      bool                  `json:"debug"`
	InsecureSkipVerify                         bool                  `json:"insecure-skip-verify"`
	RedhatSSORealm                             *IAMRealmConfig       `json:"redhat_sso_config"`
	MaxAllowedServiceAccounts                  int                   `json:"max_allowed_service_accounts"`
	MaxLimitForGetClients                      int                   `json:"max_limit_for_get_clients"`
	ServiceAccounttLimitCheckSkipOrgIDListFile string                `json:"-"`
	ServiceAccounttLimitCheckSkipOrgIDList     []string              `json:"-"`
	AdditionalSSOIssuers                       *AdditionalSSOIssuers `json:"-"`
}

// AdditionalSSOIssuers ...
type AdditionalSSOIssuers struct {
	URIs     []string
	JWKSURIs []string
	File     string
	Enabled  bool
}

// IAMRealmConfig ...
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

// NewIAMConfig ...
func NewIAMConfig() *IAMConfig {
	kc := &IAMConfig{
		SsoBaseURL:            "https://sso.redhat.com",
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
		MaxAllowedServiceAccounts:                  50,
		ServiceAccounttLimitCheckSkipOrgIDListFile: "config/service-account-limits-check-skip-org-id-list.yaml",
		AdditionalSSOIssuers:                       &AdditionalSSOIssuers{},
	}
	return kc
}

// AddFlags ...
func (ic *IAMConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&ic.BaseURL, "sso-base-url", ic.BaseURL, "The base URL of the sso, integration by default")
	fs.BoolVar(&ic.Debug, "sso-debug", ic.Debug, "Debug flag for IAM API")
	fs.BoolVar(&ic.InsecureSkipVerify, "sso-insecure", ic.InsecureSkipVerify, "Disable tls verification with sso")
	fs.IntVar(&ic.MaxAllowedServiceAccounts, "max-allowed-service-accounts", ic.MaxAllowedServiceAccounts, "Max allowed service accounts per org")
	fs.IntVar(&ic.MaxLimitForGetClients, "max-limit-for-sso-get-clients", ic.MaxLimitForGetClients, "Max limits for SSO get clients")
	fs.StringVar(&ic.RedhatSSORealm.ClientIDFile, "redhat-sso-client-id-file", ic.RedhatSSORealm.ClientIDFile, "File containing IAM privileged account client-id that has access to the OSD Cluster IDP realm")
	fs.StringVar(&ic.RedhatSSORealm.ClientSecretFile, "redhat-sso-client-secret-file", ic.RedhatSSORealm.ClientSecretFile, "File containing IAM privileged account client-secret that has access to the OSD Cluster IDP realm")
	fs.StringVar(&ic.SsoBaseURL, "redhat-sso-base-url", ic.SsoBaseURL, "The base URL of the SSO, integration by default")
	fs.StringVar(&ic.ServiceAccounttLimitCheckSkipOrgIDListFile, "service-account-limits-check-skip-org-id-list-file", ic.ServiceAccounttLimitCheckSkipOrgIDListFile, "File containing a list of Org IDs for which service account limits check will be skipped")
	fs.BoolVar(&ic.AdditionalSSOIssuers.Enabled, "enable-additional-sso-issuers", ic.AdditionalSSOIssuers.Enabled, "Enable additional SSO issuer URIs for verifying tokens")
	fs.StringVar(&ic.AdditionalSSOIssuers.File, "additional-sso-issuers-file", ic.AdditionalSSOIssuers.File, "File containing a list of SSO issuer URIs to include for verifying tokens")
}

// ReadFiles ...
func (ic *IAMConfig) ReadFiles() error {
	err := shared.ReadFileValueString(ic.RedhatSSORealm.ClientIDFile, &ic.RedhatSSORealm.ClientID)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(ic.RedhatSSORealm.ClientSecretFile, &ic.RedhatSSORealm.ClientSecret)
	if err != nil {
		return err
	}

	// Read the service account limits check skip org ID yaml file
	err = shared.ReadYamlFile(ic.ServiceAccounttLimitCheckSkipOrgIDListFile, &ic.ServiceAccounttLimitCheckSkipOrgIDList)
	if err != nil {
		if os.IsNotExist(err) {
			glog.V(10).Infof("Specified service account limits skip org IDs  file %q does not exist. Proceeding as if no service account org ID skip list was provided", ic.ServiceAccounttLimitCheckSkipOrgIDListFile)
		} else {
			return err
		}
	}

	ic.RedhatSSORealm.setDefaultURIs(ic.SsoBaseURL)

	// Read the additional issuers file. This will add additional SSO issuer URIs which shall be used as valid issuers
	// for tokens, i.e. sso.stage.redhat.com.
	if ic.AdditionalSSOIssuers.Enabled {
		err = readAdditionalIssuersFile(ic.AdditionalSSOIssuers.File, ic.AdditionalSSOIssuers)
		if err != nil {
			if os.IsNotExist(err) {
				glog.V(10).Infof("Specified additional SSO issuers file %q does not exist. "+
					"Proceeding as if no additional SSO issuers list was provided", ic.AdditionalSSOIssuers.File)
			} else {
				return err
			}
		}
		if err := ic.AdditionalSSOIssuers.resolveURIs(); err != nil {
			return err
		}
	}

	return nil
}

const (
	openidConfigurationPath = "/.well-known/openid-configuration"
)

type openIDConfiguration struct {
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

func getOpenIDConfiguration(c http.Client, baseURL string) (*openIDConfiguration, error) {
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
	var cfg openIDConfiguration
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
