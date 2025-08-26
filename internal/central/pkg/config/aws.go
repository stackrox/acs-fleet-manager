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
}

// NewAWSConfig ...
func NewAWSConfig() *AWSConfig {
	return &AWSConfig{
		AccountIDFile:       "secrets/aws.accountid",
		AccessKeyFile:       "secrets/aws.accesskey",
		SecretAccessKeyFile: "secrets/aws.secretaccesskey", // pragma: allowlist secret
	}
}

// AddFlags ...
func (c *AWSConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.AccountIDFile, "aws-account-id-file", c.AccountIDFile, "File containing AWS account id")
	fs.StringVar(&c.AccessKeyFile, "aws-access-key-file", c.AccessKeyFile, "File containing AWS access key")
	fs.StringVar(&c.SecretAccessKeyFile, "aws-secret-access-key-file", c.SecretAccessKeyFile, "File containing AWS secret access key")
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
	return nil
}
