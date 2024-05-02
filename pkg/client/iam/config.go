package iam

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"github.com/stackrox/rox/pkg/netutil"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// IAMConfig ...
type IAMConfig struct {
	JwksURL              string
	JwksFile             string
	SsoBaseURL           string
	InternalSsoBaseURL   string
	RedhatSSORealm       *IAMRealmConfig
	InternalSSORealm     *IAMRealmConfig
	AdditionalSSOIssuers *OIDCIssuers
	DataPlaneOIDCIssuers *OIDCIssuers
}

// OIDCIssuers is a list of issuers that the Fleet Manager server trusts.
type OIDCIssuers struct {
	// URIs the list of issuer uris
	URIs []string
	// JWKSURIs the list of JWKSs uris derived from URIs
	JWKSURIs []string
	// File location of the file from which the URIs will be read
	File string
	// Enabled add to the server configuration if true
	Enabled bool
}

// GetURIs returns copy of URIs to protect config from modifications.
func (a *OIDCIssuers) GetURIs() []string {
	uris := make([]string, 0, len(a.URIs))
	copy(uris, a.URIs)
	return uris
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

// IsConfigured is set to true in case client credentials are properly set.
func (c *IAMRealmConfig) IsConfigured() bool {
	return c.ClientID != ""
}

func (c *IAMRealmConfig) validateConfiguration() error {
	if !c.IsConfigured() {
		return nil
	}
	validatedFields := map[string]string{
		"clientId":         c.ClientID,
		"clientSecret":     c.ClientSecret, // pragma: allowlist secret
		"baseURL":          c.BaseURL,
		"realm":            c.Realm,
		"tokenEndpointURI": c.TokenEndpointURI,
		"validIssuerURI":   c.ValidIssuerURI,
		"apiEndpointURI":   c.APIEndpointURI,
	}
	for fieldName, fieldValue := range validatedFields {
		if fieldValue == "" {
			return fmt.Errorf("%s is empty", fieldName)
		}
	}
	if c.GrantType != "client_credentials" {
		return fmt.Errorf("grant type %q is not supported", c.GrantType)
	}
	return nil
}

// NewIAMConfig ...
func NewIAMConfig() *IAMConfig {
	kc := &IAMConfig{
		JwksURL:    "https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/certs",
		JwksFile:   "config/jwks-file.json",
		SsoBaseURL: "https://sso.redhat.com",
		RedhatSSORealm: &IAMRealmConfig{
			APIEndpointURI:   "/auth/realms/redhat-external",
			Realm:            "redhat-external",
			ClientIDFile:     "secrets/redhatsso-service.clientId",
			ClientSecretFile: "secrets/redhatsso-service.clientSecret", // pragma: allowlist secret
			GrantType:        "client_credentials",
		},
		InternalSSORealm: &IAMRealmConfig{
			APIEndpointURI: "/auth/realms/EmployeeIDP",
			Realm:          "EmployeeIDP",
		},
		InternalSsoBaseURL:   "https://auth.redhat.com",
		AdditionalSSOIssuers: &OIDCIssuers{},
		DataPlaneOIDCIssuers: &OIDCIssuers{
			Enabled: true,
			File:    "config/dataplane-oidc-issuers.yaml",
		},
	}
	return kc
}

// AddFlags ...
func (ic *IAMConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&ic.JwksURL, "jwks-url", ic.JwksURL, "The URL of the JSON web token signing certificates.")
	fs.StringVar(&ic.JwksFile, "jwks-file", ic.JwksFile, "File containing the the JSON web token signing certificates.")
	fs.StringVar(&ic.RedhatSSORealm.ClientIDFile, "redhat-sso-client-id-file", ic.RedhatSSORealm.ClientIDFile, "File containing IAM privileged account client-id that has access to the OSD Cluster IDP realm")
	fs.StringVar(&ic.RedhatSSORealm.ClientSecretFile, "redhat-sso-client-secret-file", ic.RedhatSSORealm.ClientSecretFile, "File containing IAM privileged account client-secret that has access to the OSD Cluster IDP realm")
	fs.StringVar(&ic.SsoBaseURL, "redhat-sso-base-url", ic.SsoBaseURL, "The base URL of the SSO, integration by default")
	fs.StringVar(&ic.InternalSsoBaseURL, "internal-sso-base-url", ic.InternalSsoBaseURL, "The base URL of the internal SSO, production by default")
	fs.BoolVar(&ic.AdditionalSSOIssuers.Enabled, "enable-additional-sso-issuers", ic.AdditionalSSOIssuers.Enabled, "Enable additional SSO issuer URIs for verifying tokens")
	fs.StringVar(&ic.AdditionalSSOIssuers.File, "additional-sso-issuers-file", ic.AdditionalSSOIssuers.File, "File containing a list of SSO issuer URIs to include for verifying tokens")
	fs.StringVar(&ic.DataPlaneOIDCIssuers.File, "dataplane-oidc-issuers-file", ic.DataPlaneOIDCIssuers.File, "File containing a list of OIDC issuer URIs to include for verifying tokens")
}

// ReadFiles ...
func (ic *IAMConfig) ReadFiles() error {
	ic.JwksFile = shared.BuildFullFilePath(ic.JwksFile)

	err := shared.ReadFileValueString(ic.RedhatSSORealm.ClientIDFile, &ic.RedhatSSORealm.ClientID)
	if err != nil {
		return fmt.Errorf("reading Red Hat SSO Realm ClientID file %q: %w", ic.RedhatSSORealm.ClientIDFile, err)
	}
	err = shared.ReadFileValueString(ic.RedhatSSORealm.ClientSecretFile, &ic.RedhatSSORealm.ClientSecret)
	if err != nil {
		return fmt.Errorf("reading Red Hat SSO Real Client secret file %q: %w", ic.RedhatSSORealm.ClientSecretFile, err)
	}

	ic.RedhatSSORealm.setDefaultURIs(ic.SsoBaseURL)
	ic.InternalSSORealm.setDefaultURIs(ic.InternalSsoBaseURL)
	if err := ic.RedhatSSORealm.validateConfiguration(); err != nil {
		return fmt.Errorf("validating external RH SSO realm config: %w", err)
	}
	// Internal SSO realm will not be configured with client credentials at the moment.
	// It will only serve as a configuration of the endpoints + realm.
	if err := ic.InternalSSORealm.validateConfiguration(); err != nil {
		return fmt.Errorf("validating internal RH SSO realm config: %w", err)
	}
	// Read the additional issuers file. This will add additional SSO issuer URIs which shall be used as valid issuers
	// for tokens, i.e. sso.stage.redhat.com.
	if ic.AdditionalSSOIssuers.Enabled {
		err = readIssuersFile(ic.AdditionalSSOIssuers.File, ic.AdditionalSSOIssuers)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
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
	if err := readIssuersFile(ic.DataPlaneOIDCIssuers.File, ic.DataPlaneOIDCIssuers); err != nil {
		return err
	}
	return ic.DataPlaneOIDCIssuers.resolveURIs()
}

const (
	openidConfigurationPath = "/.well-known/openid-configuration"
	kubernetesIssuer        = "https://kubernetes.default.svc"
)

type openIDConfiguration struct {
	JwksURI string `json:"jwks_uri"`
}

// resolveURIs will set the jwks URIs by taking the issuer URI and fetching the openid-configuration, setting the
// jwks URI dynamically
func (a *OIDCIssuers) resolveURIs() error {
	jwksURIs := make([]string, 0, len(a.URIs))
	for _, issuerURI := range a.URIs {
		client, err := createHTTPClient(issuerURI)
		if err != nil {
			return err
		}
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

func createHTTPClient(url string) (*http.Client, error) {
	// Special case for dev/test environments: Fleet Manager runs on the Data Plane cluster
	if url == kubernetesIssuer {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("create in-cluster k8s config: %w", err)
		}
		client, err := rest.HTTPClientFor(config)
		if err != nil {
			return nil, fmt.Errorf("create http client for in-cluster k8s issuer: %w", err)
		}
		return client, nil
	}
	// Special case for local dev environments: Fleet Manager manages a local cluster, assuming kubeconfig exists
	if isLocalCluster(url) {
		kubeconfig := os.Getenv("KUBECONFIG")
		if len(kubeconfig) == 0 {
			kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube/config")
		}
		config, err := clientcmd.BuildConfigFromFlags(url, kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("create local cluster k8s config: %w", err)
		}
		client, err := rest.HTTPClientFor(config)
		if err != nil {
			return nil, fmt.Errorf("create http client for local k8s issuer")
		}
		return client, nil
	}
	// default client
	return &http.Client{
		Timeout: time.Minute,
	}, nil
}

func isLocalCluster(uri string) bool {
	url, err := url.Parse(uri)
	if err != nil {
		glog.V(10).Infof("Unable to parse the issuer URI %v, consider it a non-local cluster", uri)
		return false
	}
	return netutil.IsLocalHost(url.Hostname())
}

func getOpenIDConfiguration(c *http.Client, baseURL string) (*openIDConfiguration, error) {
	url := strings.TrimRight(baseURL, "/") + openidConfigurationPath
	resp, err := c.Get(url)
	if err != nil {
		return nil, fmt.Errorf("executing HTTP GET request for URL %q: %w", url, err)
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
		return nil, fmt.Errorf("unmarshalling json: %w", err)
	}
	return &cfg, nil
}

func readIssuersFile(file string, endpoints *OIDCIssuers) error {
	var issuers []string
	if err := shared.ReadYamlFile(file, &issuers); err != nil {
		return fmt.Errorf("reading from yaml file: %w", err)
	}
	endpoints.URIs = issuers
	return nil
}
