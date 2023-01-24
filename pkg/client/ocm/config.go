package ocm

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

// MockModeStubServer ...
const (
	MockModeStubServer            = "stub-server"
	MockModeEmulateServer         = "emulate-server"
	centralOperatorAddonID        = "managed-central"
	fleetshardAddonID             = "fleetshard-operator"
	ClusterLoggingOperatorAddonID = "cluster-logging-operator"
)

// OCMConfig ...
type OCMConfig struct {
	BaseURL                string `json:"base_url"`
	AmsURL                 string `json:"ams_url"`
	ClientID               string `json:"client-id"`
	ClientIDFile           string `json:"client-id_file"`
	ClientSecret           string `json:"client-secret"`
	ClientSecretFile       string `json:"client-secret_file"`
	SelfToken              string `json:"self_token"`
	SelfTokenFile          string `json:"self_token_file"`
	TokenURL               string `json:"token_url"`
	Debug                  bool   `json:"debug"`
	EnableMock             bool   `json:"enable_mock"`
	MockMode               string `json:"mock_type"`
	CentralOperatorAddonID string `json:"central_operator_addon_id"`
	FleetshardAddonID      string `json:"fleetshard_addon_id"`
}

// NewOCMConfig ...
func NewOCMConfig() *OCMConfig {
	return &OCMConfig{
		BaseURL:                "https://api-integration.6943.hive-integration.openshiftapps.com",
		AmsURL:                 "https://api.stage.openshift.com",
		TokenURL:               "https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/token",
		ClientIDFile:           "secrets/ocm-service.clientId",
		ClientSecretFile:       "secrets/ocm-service.clientSecret", // pragma: allowlist secret
		SelfTokenFile:          "secrets/ocm-service.token",
		Debug:                  false,
		EnableMock:             false,
		MockMode:               MockModeStubServer,
		CentralOperatorAddonID: centralOperatorAddonID,
		FleetshardAddonID:      fleetshardAddonID,
	}
}

// AddFlags ...
func (c *OCMConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.ClientIDFile, "ocm-client-id-file", c.ClientIDFile, "File containing OCM API privileged account client-id")
	fs.StringVar(&c.ClientSecretFile, "ocm-client-secret-file", c.ClientSecretFile, "File containing OCM API privileged account client-secret")
	fs.StringVar(&c.SelfTokenFile, "self-token-file", c.SelfTokenFile, "File containing OCM API privileged offline SSO token")
	fs.StringVar(&c.BaseURL, "ocm-base-url", c.BaseURL, "The base URL of the OCM API, integration by default")
	fs.StringVar(&c.AmsURL, "ams-base-url", c.AmsURL, "The base URL of the AMS API, integration by default")
	fs.StringVar(&c.TokenURL, "ocm-token-url", c.TokenURL, "The base URL that OCM uses to request tokens, stage by default")
	fs.BoolVar(&c.Debug, "ocm-debug", c.Debug, "Debug flag for OCM API")
	fs.BoolVar(&c.EnableMock, "enable-ocm-mock", c.EnableMock, "Enable mock ocm clients")
	fs.StringVar(&c.MockMode, "ocm-mock-mode", c.MockMode, "Set mock type")
	fs.StringVar(&c.CentralOperatorAddonID, "central-operator-addon-id", c.CentralOperatorAddonID, "The name of the Central operator addon")
	fs.StringVar(&c.FleetshardAddonID, "fleetshard-addon-id", c.FleetshardAddonID, "The name of the fleetshard operator addon")
}

// ReadFiles ...
func (c *OCMConfig) ReadFiles() error {
	if c.EnableMock {
		return nil
	}

	err := shared.ReadFileValueString(c.ClientIDFile, &c.ClientID)
	if err != nil {
		return fmt.Errorf("reading client ID file: %w", err)
	}
	err = shared.ReadFileValueString(c.ClientSecretFile, &c.ClientSecret)
	if err != nil {
		return fmt.Errorf("reading client secret file: %w", err)
	}
	err = shared.ReadFileValueString(c.SelfTokenFile, &c.SelfToken)
	if err != nil && (c.ClientSecret == "" || c.ClientID == "") {
		return fmt.Errorf("reading self token file: %w", err)
	}

	return nil
}
