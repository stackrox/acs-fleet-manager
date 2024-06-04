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
	KubernetesIssuer     *KubernetesIssuer
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

// KubernetesIssuer specific to service account issuer discovery.
// Unlike OIDCIssuers used in cases where unauthorised/anonymous access to the issuer endpoint is not permitted.
// The purpose of this issuer is to support local k8s clusters.
// see: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-issuer-discovery.
type KubernetesIssuer struct {
	// Enabled add to the server configuration if true
	Enabled bool
	// IssuerURI - uri of the issuer endpoint
	IssuerURI string
	// JWKSFile location of the file to which the JWKs content is written
	JWKSFile string
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
		KubernetesIssuer: &KubernetesIssuer{
			Enabled:   true,
			IssuerURI: kubernetesIssuer,
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
	fs.BoolVar(&ic.KubernetesIssuer.Enabled, "kubernetes-issuer-enabled", ic.KubernetesIssuer.Enabled, "Enables kubernetes issuer for verifying service account tokens. Use it ONLY when the cluster issuer URI is NOT public. Otherwise, use dataplane-oidc-issuers-file instead")
	fs.StringVar(&ic.KubernetesIssuer.IssuerURI, "kubernetes-issuer-uri", ic.KubernetesIssuer.IssuerURI, "Kubernetes issuer URIs for verifying service account tokens")
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
	if len(ic.DataPlaneOIDCIssuers.JWKSURIs) == 0 {
		if err := ic.DataPlaneOIDCIssuers.resolveURIs(); err != nil {
			return err
		}
	}
	if ic.KubernetesIssuer.Enabled {
		if err := ic.KubernetesIssuer.writeJwksFile(); err != nil {
			glog.Errorf("Failed to create a temp JWKS file, skipping: %v", err)
		}
	}
	return nil
}

// GetDataPlaneIssuerURIs returns data plane issuer URIs configured for the service account token validation
func (ic *IAMConfig) GetDataPlaneIssuerURIs() []string {
	uris := make([]string, len(ic.DataPlaneOIDCIssuers.URIs))
	copy(uris, ic.DataPlaneOIDCIssuers.URIs)
	if ic.KubernetesIssuer.Enabled {
		uris = append(uris, ic.KubernetesIssuer.IssuerURI)
	}
	return uris
}

const (
	openidConfigurationPath = "/.well-known/openid-configuration"
	kubernetesIssuer        = "https://kubernetes.default.svc"
	jwksTempFilePattern     = "k8s-jwks-*.json"
)

type openIDConfiguration struct {
	Issuer  string `json:"issuer"`
	JwksURI string `json:"jwks_uri"`
}

// resolveURIs will set the jwks URIs by taking the issuer URI and fetching the openid-configuration, setting the
// jwks URI dynamically
func (a *OIDCIssuers) resolveURIs() error {
	jwksURIs := make([]string, 0, len(a.URIs))
	client := &http.Client{
		Timeout: time.Minute,
	}
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

func (i *KubernetesIssuer) writeJwksFile() error {
	jwks, err := i.fetchJwks()
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}
	tempFile, err := os.CreateTemp("", jwksTempFilePattern)
	if err != nil {
		return fmt.Errorf("create temp jwks file for a k8s issuer: %w", err)
	}
	defer tempFile.Close()
	_, err = tempFile.Write(jwks)
	if err != nil {
		return fmt.Errorf("write jwks to the temp file %v: %w", tempFile.Name(), err)
	}
	i.JWKSFile = tempFile.Name()
	glog.V(5).Infof("Wrote JWKs to the temp file %v", i.JWKSFile)
	return nil
}

func (i *KubernetesIssuer) fetchJwks() ([]byte, error) {
	client, err := i.createHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("create http client for k8s issuer %s: %w", i.IssuerURI, err)
	}
	jwksURI, err := i.getJwksURI(client)
	if err != nil {
		return nil, fmt.Errorf("get JWKS URI for k8s issuer %s: %w", i.IssuerURI, err)
	}
	resp, err := client.Get(jwksURI)
	if err != nil {
		return nil, fmt.Errorf("fetch jwks from k8s issuer %s: %w", i.IssuerURI, err)
	}
	defer resp.Body.Close()
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading JWKS response: %w", err)
	}
	return bytes, nil
}

func (i *KubernetesIssuer) createHTTPClient() (*http.Client, error) {
	config, err := i.buildK8sConfig()
	if err != nil {
		return nil, fmt.Errorf("build k8s config for issuer %s: %w", i.IssuerURI, err)
	}
	client, err := rest.HTTPClientFor(config)
	if err != nil {
		return nil, fmt.Errorf("create http client for k8s issuer %s: %w", i.IssuerURI, err)
	}
	return client, nil
}

func (i *KubernetesIssuer) getJwksURI(client *http.Client) (string, error) {
	cfg, err := getOpenIDConfiguration(client, i.IssuerURI)
	if err != nil {
		return "", errors.Wrapf(err, "retrieving open-id configuration for %q", i.IssuerURI)
	}
	jwksURI := cfg.JwksURI
	if i.isLocalCluster() {
		// kube api-server returns an internal IP, need to override it.
		jwksURI = i.overrideJwksURIForLocalCluster(jwksURI)
	}
	if cfg.Issuer != i.IssuerURI {
		glog.V(5).Infof("Configured issuer URI does't match the issuer URI configured in the discovery document, overriding: [configured: %s, got: %s]", i.IssuerURI, cfg.Issuer)
		i.IssuerURI = cfg.Issuer
	}
	return jwksURI, nil
}

func (i *KubernetesIssuer) overrideJwksURIForLocalCluster(jwksURI string) string {
	jwksURL, err := url.Parse(jwksURI)
	if err != nil {
		glog.Errorf("Failed to override JWKs URL of the local k8s cluster %s: %v", jwksURI, err)
		return jwksURI
	}
	if jwksURL.Port() != "" {
		jwksURL.Host = "127.0.0.1:" + jwksURL.Port()
	} else {
		jwksURL.Host = "127.0.0.1"
	}
	return jwksURL.String()
}

func (i *KubernetesIssuer) buildK8sConfig() (*rest.Config, error) {
	// Special case for local dev environments: Fleet Manager manages local cluster, assuming kubeconfig exists
	if i.isLocalCluster() {
		kubeconfig := os.Getenv("KUBECONFIG")
		if len(kubeconfig) == 0 {
			kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube/config")
		}
		config, err := clientcmd.BuildConfigFromFlags(i.IssuerURI, kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("create local cluster k8s config: %w", err)
		}
		return config, nil
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("create in-cluster k8s config: %w", err)
	}
	return config, nil
}

func (i *KubernetesIssuer) isLocalCluster() bool {
	issuerURL, err := url.Parse(i.IssuerURI)
	if err != nil {
		glog.V(10).Infof("Unable to parse the issuer URI %v, consider it a non-local cluster", i.IssuerURI)
		return false
	}
	return netutil.IsLocalHost(issuerURL.Hostname())
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
