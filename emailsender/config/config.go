// Package config for email sender service
package config

import (
	"fmt"

	"github.com/caarlos0/env/v6"
	"github.com/golang-jwt/jwt/v4"
	"gopkg.in/yaml.v2"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

const defaultSATokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"

// Config contains this application's runtime configuration.
type Config struct {
	ClusterID                    string `env:"CLUSTER_ID"`
	ServerAddress                string `env:"SERVER_ADDRESS" envDefault:":8080"`
	EnableHTTPS                  bool   `env:"ENABLE_HTTPS" envDefault:"false"`
	HTTPSCertFile                string `env:"HTTPS_CERT_FILE" envDefault:""`
	HTTPSKeyFile                 string `env:"HTTPS_KEY_FILE" envDefault:""`
	MetricsAddress               string `env:"METRICS_ADDRESS" envDefault:":9090"`
	AuthConfigFile               string `env:"AUTH_CONFIG_FILE" envDefault:"config/emailsender-authz.yaml"`
	AuthConfigFromServiceAccount bool   `env:"AUTH_CONFIG_FROM_SERVICE_ACCOUNT" envDefault:"true"`
	AuthConfig                   AuthConfig
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

	auth := &AuthConfig{file: c.AuthConfigFile, saTokenFile: defaultSATokenFile}
	var authError error
	if c.AuthConfigFromServiceAccount {
		authError = auth.readFromSA()
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

// AuthConfig is the configuration for authn/authz for the emailsender
type AuthConfig struct {
	file             string
	saTokenFile      string
	JwksURLs         []string `yaml:"jwks_urls"`
	AllowedIssuer    []string `yaml:"allowed_issuers"`
	AllowedOrgIDs    []string `yaml:"allowed_org_ids"`
	AllowedAudiences []string `yaml:"allowed_audiences"`
}

// readFile reads the config
func (c *AuthConfig) readFile() error {
	fileContents, err := shared.ReadFile(c.file)
	if err != nil {
		return fmt.Errorf("failed to read emailsender authz config: %w", err)
	}

	err = yaml.UnmarshalStrict([]byte(fileContents), c)
	if err != nil {
		return fmt.Errorf("failed to unmarshal emailsender authz config: %w", err)
	}

	return nil
}

// readFromSA reads the file given as saTokenFile and uses the claims
// to setup the AuthConfig. It is used for service account authentication of
// tenants to the emailsender. tenants are running in the same cluster as
// emailsender, thus their token issuer and keys must match.
// It expects a jwks file to be available at URL %iss/keys.jsons
func (c *AuthConfig) readFromSA() error {
	tokenBytes, err := shared.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return fmt.Errorf("failed to read service account token from file %w", err)
	}

	// we are parsing unverified here since injecting a service account token into the file above
	// that has invalid information requires rights that would anyway allow to modify the
	// configuration of this service including AuthConfig.
	token, _, err := jwt.NewParser().ParseUnverified(string(tokenBytes), jwt.MapClaims{})
	if err != nil {
		return fmt.Errorf("failed to parse service account token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("token does not include MapClaims")
	}

	issuer, ok := claims["iss"].(string)
	if !ok {
		return fmt.Errorf("issuer claim missing form token claims")
	}

	c.AllowedIssuer = []string{issuer}
	c.AllowedAudiences = []string{issuer}
	c.JwksURLs = []string{fmt.Sprintf("%s/keys.json", issuer)}

	return nil
}
