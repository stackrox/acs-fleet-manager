package environments

import (
	"os"

	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
)

// TestingEnvLoader ...
type TestingEnvLoader struct{}

var _ environments.EnvLoader = TestingEnvLoader{}

// NewTestingEnvLoader ...
func NewTestingEnvLoader() environments.EnvLoader {
	return TestingEnvLoader{}
}

// Defaults ...
func (t TestingEnvLoader) Defaults() map[string]string {
	return map[string]string{}
}

// ModifyConfiguration The testing environment is specifically for automated testing
// Mocks are loaded by default.
// The environment is expected to be modified as needed
func (t TestingEnvLoader) ModifyConfiguration(env *environments.Env) error {
	// Support a one-off env to allow enabling db debug in testing

	var databaseConfig *db.DatabaseConfig
	env.MustResolveAll(&databaseConfig)

	if os.Getenv("DB_DEBUG") == "true" {
		databaseConfig.Debug = true
	}
	return nil
}
