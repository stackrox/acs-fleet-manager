// Package config ...
package config

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/errorhelpers"

	"github.com/caarlos0/env/v11"
	"github.com/pkg/errors"
)

const (
	// EnvDev is the expected value of the environment variable "ENVIRONMENT" for dev deployments of fleetshard-sync
	EnvDev = "dev"
)

// Config contains this application's runtime configuration.
type Config struct {
	FleetManagerEndpoint    string        `env:"FLEET_MANAGER_ENDPOINT" envDefault:"http://127.0.0.1:8000"`
	StartupTimeout          time.Duration `env:"STARTUP_TIMEOUT" envDefault:"300s"`
	ClusterID               string        `env:"CLUSTER_ID"`
	ClusterName             string        `env:"CLUSTER_NAME"`
	Environment             string        `env:"ENVIRONMENT"`
	RuntimePollPeriod       time.Duration `env:"RUNTIME_POLL_PERIOD" envDefault:"5s"`
	AuthType                string        `env:"AUTH_TYPE" envDefault:"SERVICE_ACCOUNT_TOKEN"`
	StaticToken             string        `env:"STATIC_TOKEN"`
	ServiceAccountTokenFile string        `env:"FLEET_MANAGER_TOKEN_FILE"`
	CreateAuthProvider      bool          `env:"CREATE_AUTH_PROVIDER" envDefault:"false"`
	MetricsAddress          string        `env:"FLEETSHARD_METRICS_ADDRESS" envDefault:":8080"`
	DefaultBaseCRDURL       string        `env:"DEFAULT_BASE_CRD_URL" envDefault:"https://raw.githubusercontent.com/stackrox/stackrox/%s/operator/bundle/manifests/"`
	// TenantImagePullSecret can be used to inject a Kubernetes image pull secret into tenant namespaces.
	// If it is empty, nothing is injected (for example, it is not required when running on OpenShift).
	// It is required when central images need to fetched from a private Quay registry.
	// It needs to given as Docker Config JSON object.
	TenantImagePullSecret                      string `env:"TENANT_IMAGE_PULL_SECRET"`
	TenantDefaultArgoCdAppSourceRepoURL        string `env:"TENANT_DEFAULT_ARGOCD_APP_SOURCE_REPO_URL" envDefault:"https://github.com/stackrox/acscs-manifests.git"`
	TenantDefaultArgoCdAppSourceTargetRevision string `env:"TENANT_DEFAULT_ARGOCD_APP_SOURCE_TARGET_REVISION" envDefault:"HEAD"`
	TenantDefaultArgoCdAppSourcePath           string `env:"TENANT_DEFAULT_ARGOCD_APP_SOURCE_PATH" envDefault:"tenant-resources"`
	ArgoCdNamespace                            string `env:"ARGOCD_NAMESPACE" envDefault:"openshift-gitops"`
	ManagedDB                                  ManagedDB
	Telemetry                                  Telemetry
	AuditLogging                               AuditLogging
	SecretEncryption                           SecretEncryption
	FleetshardAddonName                        string `env:"FLEETSHARD_ADDON_NAME" envDefault:"acs-fleetshard"`
}

// ManagedDB for configuring managed DB specific parameters
type ManagedDB struct {
	Enabled             bool           `env:"MANAGED_DB_ENABLED" envDefault:"false"`
	SecurityGroup       string         `env:"MANAGED_DB_SECURITY_GROUP"`
	SubnetGroup         string         `env:"MANAGED_DB_SUBNET_GROUP"`
	PerformanceInsights bool           `env:"MANAGED_DB_PERFORMANCE_INSIGHTS" envDefault:"false"`
	SharedTags          []ManagedDBTag `envPrefix:"MANAGED_DB_TAGS"`
}

// ManagedDBTag configures shared managed DB tags
type ManagedDBTag struct {
	Key   string `env:"KEY"`
	Value string `env:"VALUE"`
}

// AuditLogging defines the parameter of the audit logging target.
type AuditLogging struct {
	Enabled            bool   `env:"AUDIT_LOG_ENABLED" envDefault:"false"`
	URLScheme          string `env:"AUDIT_LOG_URL_SCHEME" envDefault:"https"`
	AuditLogTargetHost string `env:"AUDIT_LOG_HOST" envDefault:"audit-logs-aggregator.rhacs-audit-logs"`
	AuditLogTargetPort int    `env:"AUDIT_LOG_PORT" envDefault:"8888"`
	SkipTLSVerify      bool   `env:"AUDIT_LOG_SKIP_TLS_VERIFY" envDefault:"false"`
}

// Telemetry defines parameters for pushing telemetry to a remote storage.
type Telemetry struct {
	StorageEndpoint string `env:"TELEMETRY_STORAGE_ENDPOINT"`
	StorageKey      string `env:"TELEMETRY_STORAGE_KEY"`
}

// SecretEncryption defines parameters to configure encryption of tenant secrets.
type SecretEncryption struct {
	Type  string `env:"SECRET_ENCRYPTION_TYPE" envDefault:"local"`
	KeyID string `env:"SECRET_ENCRYPTION_KEY_ID"`
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
	validateManagedDBConfig(c, &configErrors)
	validateSecretEncryptionConfig(c, &configErrors)
	validateTenantImagePullSecrets(c, &configErrors)

	cfgErr := configErrors.ToError()
	if cfgErr != nil {
		return nil, errors.Wrap(cfgErr, "unexpected configuration settings")
	}
	return &c, nil
}

func validateManagedDBConfig(c Config, configErrors *errorhelpers.ErrorList) {
	if !c.ManagedDB.Enabled {
		return
	}
	if c.ManagedDB.SecurityGroup == "" {
		configErrors.AddError(errors.New("MANAGED_DB_ENABLED == true and MANAGED_DB_SECURITY_GROUP unset in the environment"))
	}
}

func (a *AuditLogging) Endpoint(withScheme bool) string {
	if withScheme {
		return fmt.Sprintf("%s://%s:%d", a.URLScheme, a.AuditLogTargetHost, a.AuditLogTargetPort)
	}
	return fmt.Sprintf("%s:%d", a.AuditLogTargetHost, a.AuditLogTargetPort)
}

func validateSecretEncryptionConfig(c Config, configErrors *errorhelpers.ErrorList) {
	if !isDevEnvironment(c) && c.SecretEncryption.Type == "local" {
		configErrors.AddError(errors.New("SECRET_ENCRYPTION_TYPE == local not allowed for non dev environments")) // pragma: allowlist secret
	}

	if c.SecretEncryption.Type == "kms" && c.SecretEncryption.KeyID == "" {
		configErrors.AddError(errors.New("SECRET_ENCRYPTION_TYPE == kms and SECRET_ENCRYPTION_KEY_ID unset in the environment")) // pragma: allowlist secret
	}
}

func validateTenantImagePullSecrets(c Config, configErrors *errorhelpers.ErrorList) {
	if c.TenantImagePullSecret == "" {
		return
	}

	type dockerConfig struct {
		Auths map[string]map[string]string `json:"auths,omitempty"`
	}

	var cfg dockerConfig

	if err := json.Unmarshal([]byte(c.TenantImagePullSecret), &cfg); err != nil {
		configErrors.AddError(errors.Wrapf(err, "invalid tenant image pull secret JSON"))
		return
	}

	if cfg.Auths == nil || len(cfg.Auths) == 0 {
		configErrors.AddError(errors.New("invalid tenant image pull secret"))
	}
}

func isDevEnvironment(c Config) bool {
	return c.Environment == EnvDev || c.Environment == ""
}
