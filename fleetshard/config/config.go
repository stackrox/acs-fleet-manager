// Package config ...
package config

import (
	"time"

	"github.com/stackrox/rox/pkg/errorhelpers"

	"github.com/caarlos0/env/v6"
	"github.com/pkg/errors"
)

// Config contains this application's runtime configuration.
type Config struct {
	FleetManagerEndpoint string        `env:"FLEET_MANAGER_ENDPOINT" envDefault:"http://127.0.0.1:8000"`
	ClusterID            string        `env:"CLUSTER_ID"`
	RuntimePollPeriod    time.Duration `env:"RUNTIME_POLL_PERIOD" envDefault:"5s"`
	AuthType             string        `env:"AUTH_TYPE" envDefault:"RHSSO"`
	RHSSOClientID        string        `env:"RHSSO_SERVICE_ACCOUNT_CLIENT_ID"`
	RHSSOClientSecret    string        `env:"RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET"`
	RHSSORealm           string        `env:"RHSSO_REALM" envDefault:"redhat-external"`
	RHSSOEndpoint        string        `env:"RHSSO_ENDPOINT" envDefault:"https://sso.redhat.com"`
	OCMRefreshToken      string        `env:"OCM_TOKEN"`
	StaticToken          string        `env:"STATIC_TOKEN"`
	CreateAuthProvider   bool          `env:"CREATE_AUTH_PROVIDER" envDefault:"false"`
	MetricsAddress       string        `env:"FLEETSHARD_METRICS_ADDRESS" envDefault:":8080"`
	EgressProxyImage     string        `env:"EGRESS_PROXY_IMAGE"`
	AWSRegion            string        `env:"AWS_REGION" envDefault:"us-east-1"`
	AWSRoleARN           string        `env:"AWS_ROLE_ARN"`

	ManagedDBEnabled       bool   `env:"MANAGED_DB_ENABLED" envDefault:"false"`
	ManagedDBSecurityGroup string `env:"MANAGED_DB_SECURITY_GROUP"`
	ManagedDBSubnetGroup   string `env:"MANAGED_DB_SUBNET_GROUP"`
}

// GetConfig retrieves the current runtime configuration from the environment and returns it.
func GetConfig() (*Config, error) {
	c := Config{}
	var configErrors errorhelpers.ErrorList

	if err := env.Parse(&c); err != nil {
		return nil, errors.Wrapf(err, "Unable to parse runtime configuration from environment")
	}
	if c.ClusterID == "" {
		configErrors.AddError(errors.New("CLUSTER_ID unset in the environment"))
	}
	if c.FleetManagerEndpoint == "" {
		configErrors.AddError(errors.New("FLEET_MANAGER_ENDPOINT unset in the environment"))
	}
	if c.AuthType == "" {
		configErrors.AddError(errors.New("AUTH_TYPE unset in the environment"))
	}
	validateManagedDBConfig(c, configErrors)

	cfgErr := configErrors.ToError()
	if cfgErr != nil {
		return nil, errors.Wrap(cfgErr, "unexpected configuration settings")
	}
	return &c, nil
}

func validateManagedDBConfig(c Config, configErrors errorhelpers.ErrorList) {
	if !c.ManagedDBEnabled {
		return
	}
	if c.AWSRoleARN == "" {
		configErrors.AddError(errors.New("MANAGED_DB_ENABLED == true and AWS_ROLE_ARN unset in the environment"))
	}
	if c.ManagedDBSecurityGroup == "" {
		configErrors.AddError(errors.New("MANAGED_DB_ENABLED == true and MANAGED_DB_SECURITY_GROUP unset in the environment"))
	}
}
