// Package config for email sender service
package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/caarlos0/env/v6"
	"gopkg.in/yaml.v2"

	"github.com/pkg/errors"
	commonDbConfig "github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	defaultSATokenFile      = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	defaultKubernetesCAFile = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	wellKnownPath           = ".well-known/openid-configuration"
)

// Config contains this application's runtime configuration.
type Config struct {
	ClusterID                 string        `env:"CLUSTER_ID"`
	ServerAddress             string        `env:"SERVER_ADDRESS" envDefault:":8080"`
	EnableHTTPS               bool          `env:"ENABLE_HTTPS" envDefault:"false"`
	HTTPSCertFile             string        `env:"HTTPS_CERT_FILE" envDefault:""`
	HTTPSKeyFile              string        `env:"HTTPS_KEY_FILE" envDefault:""`
	MetricsAddress            string        `env:"METRICS_ADDRESS" envDefault:":9090"`
	AuthConfigFile            string        `env:"AUTH_CONFIG_FILE" envDefault:"config/emailsender-authz.yaml"`
	AuthConfigFromKubernetes  bool          `env:"AUTH_CONFIG_FROM_KUBERNETES" envDefault:"false"`
	KubernetesSvcURL          string        `env:"KUBERNETES_SVC_URL" envDefault:"https://kubernetes.default.svc"`
	KubernetesJWKSPath        string        `env:"KUBERNETES_JWKS_PATH" envDefault:"/openid/v1/jwks"`
	SenderAddress             string        `env:"SENDER_ADDRESS" envDefault:"noreply@mail.rhacs-dev.com"`
	LimitEmailPerTenant       int           `env:"LIMIT_EMAIL_PER_TENANT" envDefault:"250"`
	SesMaxBackoffDelay        time.Duration `env:"SES_MAX_BACKOFF_DELAY" envDefault:"5s"`
	SesMaxAttempts            int           `env:"SES_MAX_ATTEMPTS" envDefault:"3"`
	EmailCleanupPeriodSeconds int           `env:"EMAIL_CLEANUP_PERIOD_SECONDS" envDefault:"300"`
	EmailCleanupExpiryDays    int           `env:"EMAIL_CLEANUP_EXPIRY_DAYS" envDefault:"2"`
	AuthConfig                AuthConfig
	DatabaseConfig            DbConfig
}

type DbConfig struct {
	HostFile           string `env:"DATABASE_HOST_FILE" envDefault:"secrets/db.host"`
	PortFile           string `env:"DATABASE_PORT_FILE" envDefault:"secrets/db.port"`
	NameFile           string `env:"DATABASE_NAME_FILE" envDefault:"secrets/db.name"`
	UserFile           string `env:"DATABASE_USER_FILE" envDefault:"secrets/db.user"`
	PasswordFile       string `env:"DATABASE_PASSWORD_FILE" envDefault:"secrets/db.password"`
	CaCertFile         string `env:"DATABASE_CA_CERT_FILE" envDefault:"secrets/db.ca_cert"`
	Host               string `env:"DATABASE_HOST" envDefault:"localhost"`
	Port               int    `env:"DATABASE_PORT" envDefault:"5432"`
	Name               string `env:"DATABASE_NAME" envDefault:"postgres"`
	User               string `env:"DATABASE_USER" envDefault:"postgres"`
	Password           string `env:"DATABASE_PASSWORD" envDefault:"postgres"`
	SSLMode            string `env:"DATABASE_SSL_MODE" envDefault:"disable"`
	MaxOpenConnections int    `env:"DATABASE_MAX_CONNECTIONS" envDefault:"50"`
}

func (d *DbConfig) GetDbConfig() *commonDbConfig.DatabaseConfig {
	cfg := commonDbConfig.NewDatabaseConfig()
	cfg.SSLMode = d.SSLMode
	cfg.MaxOpenConnections = d.MaxOpenConnections
	cfg.HostFile = d.HostFile
	cfg.PortFile = d.PortFile
	cfg.NameFile = d.NameFile
	cfg.UsernameFile = d.UserFile
	cfg.PasswordFile = d.PasswordFile // pragma: allowlist secret
	cfg.DatabaseCaCertFile = d.CaCertFile
	cfg.Host = d.Host
	cfg.Port = d.Port
	cfg.Name = d.Name
	cfg.Username = d.User
	cfg.Password = d.Password // pragma: allowlist secret
	return cfg
}

// GetConfig retrieves the current runtime configuration from the environment and returns it.
func GetConfig() (*Config, error) {
	c := Config{}
	var configErrors errorhelpers.ErrorList

	if err := env.Parse(&c); err != nil {
		return nil, errors.Wrap(err, "unable to parse runtime configuration from environment")
	}

	if c.ClusterID == "" {
		configErrors.AddError(errors.New("CLUSTER_ID environment variable is not set"))
	}

	if c.EnableHTTPS {
		if c.HTTPSCertFile == "" || c.HTTPSKeyFile == "" {
			configErrors.AddError(errors.New("ENABLE_HTTPS is true but required variables HTTPS_CERT_FILE or HTTPS_KEY_FILE are empty"))
		}
	}

	auth := &AuthConfig{
		configFile:  c.AuthConfigFile,
		saTokenFile: defaultSATokenFile,
		k8sSvcURL:   c.KubernetesSvcURL,
		k8sJWKSPath: c.KubernetesJWKSPath,
		jwksDir:     os.TempDir(),
	}

	var authError error
	if c.AuthConfigFromKubernetes {
		client, err := k8sSvcClient()
		if err != nil {
			authError = err
		} else {
			auth.httpClient = client
			authError = auth.readFromKubernetes()
		}
	} else {
		authError = auth.readFile()
	}

	if authError != nil {
		configErrors.AddError(authError)
	}

	c.AuthConfig = *auth

	if cfgErr := configErrors.ToError(); cfgErr != nil {
		return nil, errors.Wrap(cfgErr, "invalid configuration settings")
	}
	return &c, nil
}

type oidcConfig struct {
	JwksURI string `json:"jwks_uri"`
	Issuer  string `json:"issuer"`
}

// AuthConfig is the configuration for authn/authz for the emailsender
type AuthConfig struct {
	configFile       string
	saTokenFile      string
	k8sSvcURL        string
	k8sJWKSPath      string
	httpClient       *http.Client
	jwksDir          string
	JwksURLs         []string `yaml:"jwks_urls"`
	JwksFiles        []string `yaml:"jwks_files"`
	AllowedIssuer    []string `yaml:"allowed_issuers"`
	AllowedOrgIDs    []string `yaml:"allowed_org_ids"`
	AllowedAudiences []string `yaml:"allowed_audiences"`
}

// readFile reads the config
func (c *AuthConfig) readFile() error {
	fileContents, err := shared.ReadFile(c.configFile)
	if err != nil {
		return fmt.Errorf("failed to read emailsender authz config: %w", err)
	}

	err = yaml.UnmarshalStrict([]byte(fileContents), c)
	if err != nil {
		return fmt.Errorf("failed to unmarshal emailsender authz config: %w", err)
	}

	return nil
}

// readFromKubernetes uses the service account token and the Kubernetes api
// to derive an AuthConfig from the Kubernetes openid-configuration
func (c *AuthConfig) readFromKubernetes() error {
	// we need the own SA token to be able to authenticate to the jwks key endpoint
	// since we're not allowed to call it with an anonymous user
	tokenBytes, err := shared.ReadFile(c.saTokenFile)
	if err != nil {
		return fmt.Errorf("failed to read service account token from file %w", err)
	}

	token := string(tokenBytes)
	oidcCfg, err := c.getOIDCConfig(token)
	if err != nil {
		return err
	}

	jwksBytes, err := c.getJWKS(token)
	if err != nil {
		return err
	}

	jwksFilePath := path.Join(c.jwksDir, "jwks.json")
	// We store this in a file because the OCM SDK middleware we use for auth
	// isn't able to call a jwks URL that requires authentication.
	// As a workaround we can pre-load the jwks to a file and use the JwksFile option of that SDK.
	if err := os.WriteFile(jwksFilePath, jwksBytes, 0644); err != nil {
		return fmt.Errorf("failed to store jwks file in temp dir: %w", err)
	}

	// for default svc account token issuer == audience
	c.AllowedAudiences = []string{oidcCfg.Issuer}
	c.AllowedIssuer = []string{oidcCfg.Issuer}
	c.JwksFiles = []string{jwksFilePath}

	return nil
}

func (c *AuthConfig) getOIDCConfig(token string) (oidcConfig, error) {
	var oidcCfg oidcConfig

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", c.k8sSvcURL, wellKnownPath), nil)
	if err != nil {
		return oidcCfg, fmt.Errorf("failed to create HTTP request for openid configuration: %w", err)
	}
	addAuthHeader(req, token)

	oidcCfgRes, err := c.httpClient.Do(req)
	if err != nil {
		return oidcCfg, fmt.Errorf("failed to send HTTP requests for openid configuration: %w", err)
	}
	defer utils.IgnoreError(oidcCfgRes.Body.Close)

	if oidcCfgRes.StatusCode != 200 {
		return oidcCfg, fmt.Errorf("HTTP request for openid configuration failed with status: %d", oidcCfgRes.StatusCode)
	}

	if err := json.NewDecoder(oidcCfgRes.Body).Decode(&oidcCfg); err != nil {
		return oidcCfg, fmt.Errorf("failed to decoded openid configuration response body: %w", err)
	}

	return oidcCfg, nil
}

func (c *AuthConfig) getJWKS(token string) ([]byte, error) {
	jwksURL := fmt.Sprintf("%s/%s", c.k8sSvcURL, c.k8sJWKSPath)
	jwksRequest, err := http.NewRequest(http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request for jwks: %w", err)
	}
	addAuthHeader(jwksRequest, token)

	jwksRes, err := c.httpClient.Do(jwksRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request for jwks: %w", err)
	}
	defer utils.IgnoreError(jwksRes.Body.Close)

	if jwksRes.StatusCode != 200 {
		return nil, fmt.Errorf("jwks key request to %s failed with status code: %d", jwksURL, jwksRes.StatusCode)
	}

	jwksBytes, err := io.ReadAll(jwksRes.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for jwks: %w", err)
	}

	return jwksBytes, nil
}

func addAuthHeader(req *http.Request, token string) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
}

func k8sSvcClient() (*http.Client, error) {
	tlsConf, err := shared.TLSWithAdditionalCAs(defaultKubernetesCAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create tls conf: %w", err)
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConf,
		},
	}, nil
}
