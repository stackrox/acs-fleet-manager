package environments

import (
	"fmt"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
)

// GetEnvironmentLoader returns an environment loader
func GetEnvironmentLoader(name string) environments.EnvLoader {
	switch name {
	case environments.TestingEnv:
		return NewTestingEnvLoader()
	case environments.DevelopmentEnv:
		return NewDevelopmentEnvLoader()
	case environments.ProductionEnv:
		return NewProductionEnvLoader()
	case environments.StageEnv:
		return NewStageEnvLoader()
	case environments.IntegrationEnv:
		return NewIntegrationEnvLoader()
	default:
		panic(fmt.Sprintf("Environment does not exist %s", name))
	}
}
