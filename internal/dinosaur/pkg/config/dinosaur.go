package config

import (
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

type MaxCapacityConfig struct {
	MaxCapacity int64 `json:"maxCapacity"`
}

type DinosaurConfig struct {
	DinosaurTLSCert                   string `json:"dinosaur_tls_cert"`
	DinosaurTLSCertFile               string `json:"dinosaur_tls_cert_file"`
	DinosaurTLSKey                    string `json:"dinosaur_tls_key"`
	DinosaurTLSKeyFile                string `json:"dinosaur_tls_key_file"`
	EnableDinosaurExternalCertificate bool   `json:"enable_dinosaur_external_certificate"`
	DinosaurDomainName                string `json:"dinosaur_domain_name"`
	// TODO(ROX-11289): drop MaxCapacity
	MaxCapacity MaxCapacityConfig `json:"max_capacity_config"`

	DinosaurLifespan      *DinosaurLifespanConfig `json:"dinosaur_lifespan"`
	Quota                 *DinosaurQuotaConfig    `json:"dinosaur_quota"`
	RhSsoClientSecret     string                  `json:"rhsso_client_secret"`
	RhSsoClientSecretFile string                  `json:"rhsso_client_secret_file"`
}

func NewDinosaurConfig() *DinosaurConfig {
	return &DinosaurConfig{
		DinosaurTLSCertFile:               "secrets/dinosaur-tls.crt",
		DinosaurTLSKeyFile:                "secrets/dinosaur-tls.key",
		EnableDinosaurExternalCertificate: false,
		DinosaurDomainName:                "dinosaur.devshift.org",
		DinosaurLifespan:                  NewDinosaurLifespanConfig(),
		Quota:                             NewDinosaurQuotaConfig(),
	}
}

func (c *DinosaurConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.DinosaurTLSCertFile, "dinosaur-tls-cert-file", c.DinosaurTLSCertFile, "File containing dinosaur certificate")
	fs.StringVar(&c.DinosaurTLSKeyFile, "dinosaur-tls-key-file", c.DinosaurTLSKeyFile, "File containing dinosaur certificate private key")
	fs.BoolVar(&c.EnableDinosaurExternalCertificate, "enable-dinosaur-external-certificate", c.EnableDinosaurExternalCertificate, "Enable custom certificate for Dinosaur TLS")
	fs.BoolVar(&c.DinosaurLifespan.EnableDeletionOfExpiredDinosaur, "enable-deletion-of-expired-dinosaur", c.DinosaurLifespan.EnableDeletionOfExpiredDinosaur, "Enable the deletion of dinosaurs when its life span has expired")
	fs.IntVar(&c.DinosaurLifespan.DinosaurLifespanInHours, "dinosaur-lifespan", c.DinosaurLifespan.DinosaurLifespanInHours, "The desired lifespan of a Dinosaur instance")
	fs.StringVar(&c.DinosaurDomainName, "dinosaur-domain-name", c.DinosaurDomainName, "The domain name to use for Dinosaur instances")
	fs.StringVar(&c.Quota.Type, "quota-type", c.Quota.Type, "The type of the quota service to be used. The available options are: 'ams' for AMS backed implementation and 'quota-management-list' for quota list backed implementation (default).")
	fs.BoolVar(&c.Quota.AllowEvaluatorInstance, "allow-evaluator-instance", c.Quota.AllowEvaluatorInstance, "Allow the creation of dinosaur evaluator instances")
	fs.StringVar(&c.RhSsoClientSecretFile, "rhsso-client-secret-file", c.RhSsoClientSecretFile, "File containing OIDC client secret of sso.redhat.com client")
}

func (c *DinosaurConfig) ReadFiles() error {
	err := shared.ReadFileValueString(c.DinosaurTLSCertFile, &c.DinosaurTLSCert)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(c.DinosaurTLSKeyFile, &c.DinosaurTLSKey)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(c.RhSsoClientSecretFile, &c.RhSsoClientSecret)
	if err != nil {
		return err
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
