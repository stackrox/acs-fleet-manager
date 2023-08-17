package config

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

// CentralConfig ...
type CentralConfig struct {
	CentralTLSCert                   string `json:"central_tls_cert"`
	CentralTLSCertFile               string `json:"central_tls_cert_file"`
	CentralTLSKey                    string `json:"central_tls_key"`
	CentralTLSKeyFile                string `json:"central_tls_key_file"`
	EnableCentralExternalCertificate bool   `json:"enable_central_external_certificate"`
	CentralDomainName                string `json:"central_domain_name"`
	// CentralDefaultVersion is the default version for new Central instances. If this is set to a
	// non empty value, fleet-manager will store it as the new default value in its database on start up.
	CentralDefaultVersion string `json:"central_default_version"`

	CentralLifespan *CentralLifespanConfig `json:"central_lifespan"`
	Quota           *CentralQuotaConfig    `json:"central_quota"`

	// Central's IdP static configuration (optional).
	CentralIDPClientID         string `json:"central_idp_client_id"`
	CentralIDPClientSecret     string `json:"central_idp_client_secret"`
	CentralIDPClientSecretFile string `json:"central_idp_client_secret_file"`
	CentralIDPIssuer           string `json:"central_idp_issuer"`
	// CentralRetentionPeriod configures how long it should be possible to restore a central tenant
	// that has been deleted via API
	CentralRetentionPeriodDays int `json:"central_retention_period_days"`
}

// NewCentralConfig ...
func NewCentralConfig() *CentralConfig {
	return &CentralConfig{
		CentralTLSCertFile:               "secrets/central-tls.crt",
		CentralTLSKeyFile:                "secrets/central-tls.key",
		EnableCentralExternalCertificate: false,
		CentralDomainName:                "rhacs-dev.com",
		CentralLifespan:                  NewCentralLifespanConfig(),
		Quota:                            NewCentralQuotaConfig(),
		CentralIDPClientSecretFile:       "secrets/central.idp-client-secret", //pragma: allowlist secret
		CentralIDPIssuer:                 "https://sso.redhat.com/auth/realms/redhat-external",
		CentralRetentionPeriodDays:       7,
	}
}

// AddFlags ...
func (c *CentralConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.CentralTLSCertFile, "central-tls-cert-file", c.CentralTLSCertFile, "File containing central certificate")
	fs.StringVar(&c.CentralTLSKeyFile, "central-tls-key-file", c.CentralTLSKeyFile, "File containing central certificate private key")
	fs.BoolVar(&c.EnableCentralExternalCertificate, "enable-central-external-certificate", c.EnableCentralExternalCertificate, "Enable custom certificate for Central TLS")
	fs.BoolVar(&c.CentralLifespan.EnableDeletionOfExpiredCentral, "enable-deletion-of-expired-central", c.CentralLifespan.EnableDeletionOfExpiredCentral, "Enable the deletion of centrals when its life span has expired")
	fs.IntVar(&c.CentralLifespan.CentralLifespanInHours, "central-lifespan", c.CentralLifespan.CentralLifespanInHours, "The desired lifespan of a Central instance")
	fs.StringVar(&c.CentralDomainName, "central-domain-name", c.CentralDomainName, "The domain name to use for Central instances")
	fs.StringVar(&c.CentralDefaultVersion, "central-default-version", c.CentralDefaultVersion, "The default version for Central instances")
	fs.StringVar(&c.Quota.Type, "quota-type", c.Quota.Type, "The type of the quota service to be used. The available options are: 'ams' for AMS backed implementation and 'quota-management-list' for quota list backed implementation (default).")
	fs.BoolVar(&c.Quota.AllowEvaluatorInstance, "allow-evaluator-instance", c.Quota.AllowEvaluatorInstance, "Allow the creation of central evaluator instances")

	fs.StringVar(&c.CentralIDPClientID, "central-idp-client-id", c.CentralIDPClientID, "OIDC client_id to pass to Central's auth config")
	fs.StringVar(&c.CentralIDPClientSecretFile, "central-idp-client-secret-file", c.CentralIDPClientSecretFile, "File containing OIDC client_secret to pass to Central's auth config")
	fs.StringVar(&c.CentralIDPIssuer, "central-idp-issuer", c.CentralIDPIssuer, "OIDC issuer URL to pass to Central's auth config")
	fs.IntVar(&c.CentralRetentionPeriodDays, "central-retention-period-days", c.CentralRetentionPeriodDays, "The number of days after deletion until central tenants can no longer be restored")
}

// ReadFiles ...
func (c *CentralConfig) ReadFiles() error {
	err := shared.ReadFileValueString(c.CentralTLSCertFile, &c.CentralTLSCert)
	if err != nil {
		return fmt.Errorf("reading TLS certificate file: %w", err)
	}
	err = shared.ReadFileValueString(c.CentralTLSKeyFile, &c.CentralTLSKey)
	if err != nil {
		return fmt.Errorf("reading TLS key file: %w", err)
	}

	// Initialise and check that all parts of static auth config are present.
	if c.HasStaticAuth() {
		err = shared.ReadFileValueString(c.CentralIDPClientSecretFile, &c.CentralIDPClientSecret)
		if err != nil {
			return fmt.Errorf("reading Central's IdP client secret file: %w", err)
		}
		if c.CentralIDPClientSecret == "" {
			return errors.Errorf("no client_secret specified for static client_id %q;"+
				" auth configuration is either incorrect or insecure", c.CentralIDPClientID)
		}
		if c.CentralIDPIssuer == "" {
			return errors.Errorf("no issuer specified for static client_id %q;"+
				" auth configuration will likely not work properly", c.CentralIDPClientID)
		}
	}

	return nil
}

// HasStaticAuth returns true if the static auth config for Centrals has been
// specified and false otherwise.
func (c *CentralConfig) HasStaticAuth() bool {
	// We don't look at other integral parts of the auth config like
	// CentralIDPIssuer or CentralIDPClientSecret. Failure to provide a working auth
	// configuration should not mask an intent to use static configuration.
	return c.CentralIDPClientID != ""
}
