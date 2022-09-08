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
	CentralTLSCert                   string `json:"central_tls_cert"`
	CentralTLSCertFile               string `json:"central_tls_cert_file"`
	CentralTLSKey                    string `json:"central_tls_key"`
	CentralTLSKeyFile                string `json:"central_tls_key_file"`
	EnableCentralExternalCertificate bool   `json:"enable_central_external_certificate"`
	CentralDomainName                string `json:"central_domain_name"`
	// TODO(ROX-11289): drop MaxCapacity
	MaxCapacity MaxCapacityConfig `json:"max_capacity_config"`

	CentralLifespan *CentralLifespanConfig `json:"central_lifespan"`
	Quota           *CentralQuotaConfig    `json:"central_quota"`

	// Central's IdP static configuration (optional).
	CentralIdPClientID         string `json:"central_idp_client_id"`
	CentralIdPClientSecret     string `json:"central_idp_client_secret"`
	CentralIdPClientSecretFile string `json:"central_idp_client_secret_file"`
	CentralIdPIssuer           string `json:"central_idp_issuer"`
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
		CentralIdPClientSecretFile:       "secrets/central.idp-client-secret", //pragma: allowlist secret
		CentralIdPIssuer:                 "https://sso.redhat.com/auth/realms/redhat-external",
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

	fs.StringVar(&c.CentralIdPClientID, "central-idp-client-id", c.CentralIdPClientID, "OIDC client_id to pass to Central's auth config")
	fs.StringVar(&c.CentralIdPClientSecretFile, "central-idp-client-secret-file", c.CentralIdPClientSecretFile, "File containing OIDC client_secret to pass to Central's auth config")
	fs.StringVar(&c.CentralIdPIssuer, "central-idp-issuer", c.CentralIdPIssuer, "OIDC issuer URL to pass to Central's auth config")
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
	err = shared.ReadFileValueString(c.CentralIdPClientSecretFile, &c.CentralIdPClientSecret)
	if err != nil {
		return fmt.Errorf("reading Central's IdP client secret file: %w", err)
	}
	if c.CentralIdPClientSecret != "" {
		glog.Info("Central's IdP client secret is configured")
	} else {
		glog.Infof("Central's IdP client secret from file %q is missing", c.CentralIdPClientSecretFile)
	}
	// TODO(ROX-11289): drop MaxCapacity
	// MaxCapacity is deprecated and will not be used.
	// Temporarily set MaxCapacity manually in order to simplify app start.
	c.MaxCapacity = MaxCapacityConfig{1000}
	return nil
}
