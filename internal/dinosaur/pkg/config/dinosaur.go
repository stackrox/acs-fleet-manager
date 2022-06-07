package config

import (
	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

type DinosaurCapacityConfig struct {
	MaxCapacity int64 `json:"maxCapacity"`
}

type DinosaurConfig struct {
	DinosaurTLSCert                   string `json:"dinosaur_tls_cert"`
	DinosaurTLSCertFile               string `json:"dinosaur_tls_cert_file"`
	DinosaurTLSKey                    string `json:"dinosaur_tls_key"`
	DinosaurTLSKeyFile                string `json:"dinosaur_tls_key_file"`
	EnableDinosaurExternalCertificate bool   `json:"enable_dinosaur_external_certificate"`
	DinosaurDomainName                string `json:"dinosaur_domain_name"`
	// TODO drop DinosaurCapacity
	DinosaurCapacity DinosaurCapacityConfig `json:"dinosaur_capacity_config"`

	DinosaurLifespan *DinosaurLifespanConfig `json:"dinosaur_lifespan"`
	Quota            *DinosaurQuotaConfig    `json:"dinosaur_quota"`
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
	c.DinosaurCapacity = DinosaurCapacityConfig{1000}
	return nil
}
