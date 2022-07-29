package config

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

// MaxCapacityConfig ...
type MaxCapacityConfig struct {
	MaxCapacity int64 `json:"maxCapacity"`
}

// CentralConfig ...
type CentralConfig struct {
	CentralTLSCert                   string `json:"dinosaur_tls_cert"`
	CentralTLSCertFile               string `json:"dinosaur_tls_cert_file"`
	CentralTLSKey                    string `json:"dinosaur_tls_key"`
	CentralTLSKeyFile                string `json:"dinosaur_tls_key_file"`
	EnableCentralExternalCertificate bool   `json:"enable_dinosaur_external_certificate"`
	CentralDomainName                string `json:"dinosaur_domain_name"`
	// TODO(ROX-11289): drop MaxCapacity
	MaxCapacity MaxCapacityConfig `json:"max_capacity_config"`

	CentralLifespan       *CentralLifespanConfig `json:"dinosaur_lifespan"`
	Quota                 *CentralQuotaConfig    `json:"dinosaur_quota"`
	RhSsoClientSecret     string                 `json:"rhsso_client_secret"`
	RhSsoClientSecretFile string                 `json:"rhsso_client_secret_file"`
	RhSsoIssuer           string                 `json:"rhsso_issuer"`
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
		RhSsoIssuer:                      "https://sso.redhat.com/auth/realms/redhat-external",
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
	fs.StringVar(&c.Quota.Type, "quota-type", c.Quota.Type, "The type of the quota service to be used. The available options are: 'ams' for AMS backed implementation and 'quota-management-list' for quota list backed implementation (default).")
	fs.BoolVar(&c.Quota.AllowEvaluatorInstance, "allow-evaluator-instance", c.Quota.AllowEvaluatorInstance, "Allow the creation of central evaluator instances")
	fs.StringVar(&c.RhSsoClientSecretFile, "rhsso-client-secret-file", c.RhSsoClientSecretFile, "File containing OIDC client secret of sso.redhat.com client")
	fs.StringVar(&c.RhSsoIssuer, "rhsso-issuer", c.RhSsoIssuer, "Issuer identifier for sso.redhat.com. Should be equal to value returned in ID Token issuer('iss') field")
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
	err = shared.ReadFileValueString(c.RhSsoClientSecretFile, &c.RhSsoClientSecret)
	if err != nil {
		return fmt.Errorf("reading Red Hat SSO client secret file: %w", err)
	}
	if c.RhSsoClientSecret != "" {
		glog.Info("Central Red Hat OIDC client secret is configured.")
	} else {
		glog.Infof("Central Red Hat OIDC client secret from secret file %q is missing.", c.RhSsoClientSecretFile)
	}
	// TODO(ROX-11289): drop MaxCapacity
	// MaxCapacity is deprecated and will not be used.
	// Temporarily set MaxCapacity manually in order to simplify app start.
	c.MaxCapacity = MaxCapacityConfig{1000}
	return nil
}
