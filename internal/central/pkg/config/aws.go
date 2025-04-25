// Package config ...
package config

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

// AWSConfig ...
type AWSConfig struct {
	// Used for OSD Cluster creation with OCM
	AccountID           string `json:"account_id"`
	AccountIDFile       string `json:"account_id_file"`
	AccessKey           string `json:"access_key"`
	AccessKeyFile       string `json:"access_key_file"`
	SecretAccessKey     string `json:"secret_access_key"`
	SecretAccessKeyFile string `json:"secret_access_key_file"`

	// Used for domain modifications in Route 53
	Route53AccessKey           string `json:"route53_access_key"`
	Route53AccessKeyFile       string `json:"route53_access_key_file"`
	Route53SecretAccessKey     string `json:"route53_secret_access_key"`
	Route53SecretAccessKeyFile string `json:"route53_secret_access_key_file"`
}

// NewAWSConfig ...
func NewAWSConfig() *AWSConfig {
	return &AWSConfig{
		AccountIDFile:              "secrets/aws.accountid",
		AccessKeyFile:              "secrets/aws.accesskey",
		SecretAccessKeyFile:        "secrets/aws.secretaccesskey", // pragma: allowlist secret
		Route53AccessKeyFile:       "secrets/aws.route53accesskey",
		Route53SecretAccessKeyFile: "secrets/aws.route53secretaccesskey", // pragma: allowlist secret
	}
}

// AddFlags ...
func (c *AWSConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.AccountIDFile, "aws-account-id-file", c.AccountIDFile, "File containing AWS account id")
	fs.StringVar(&c.AccessKeyFile, "aws-access-key-file", c.AccessKeyFile, "File containing AWS access key")
	fs.StringVar(&c.SecretAccessKeyFile, "aws-secret-access-key-file", c.SecretAccessKeyFile, "File containing AWS secret access key")
	fs.StringVar(&c.Route53AccessKeyFile, "aws-route53-access-key-file", c.Route53AccessKeyFile, "File containing AWS access key for route53")
	fs.StringVar(&c.Route53SecretAccessKeyFile, "aws-route53-secret-access-key-file", c.Route53SecretAccessKeyFile, "File containing AWS secret access key for route53")
}

// ReadFiles ...
func (c *AWSConfig) ReadFiles() error {
	err := shared.ReadFileValueString(c.AccountIDFile, &c.AccountID)
	if err != nil {
		return fmt.Errorf("reading account ID file: %w", err)
	}
	err = shared.ReadFileValueString(c.AccessKeyFile, &c.AccessKey)
	if err != nil {
		return fmt.Errorf("reading access key file: %w", err)
	}
	err = shared.ReadFileValueString(c.SecretAccessKeyFile, &c.SecretAccessKey)
	if err != nil {
		return fmt.Errorf("reading secret access key file: %w", err)
	}
	err = shared.ReadFileValueString(c.Route53AccessKeyFile, &c.Route53AccessKey)
	if err != nil {
		return fmt.Errorf("reading route 53 access key file: %w", err)
	}
	err = shared.ReadFileValueString(c.Route53SecretAccessKeyFile, &c.Route53SecretAccessKey)
	if err != nil {
		return fmt.Errorf("reading route 53 secret access key file: %w", err)
	}
	return nil
}
