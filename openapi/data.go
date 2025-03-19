package openapi

import (
	_ "embed"
	coreHandlers "github.com/stackrox/acs-fleet-manager/pkg/handlers"
	"net/http"
	"sigs.k8s.io/yaml"
)

//go:embed fleet-manager.yaml
var fleetManagerYAML []byte
var fleetManagerJSONBytes []byte

func HandleGetFleetManagerOpenApiDefinition() http.HandlerFunc {
	return http.HandlerFunc(coreHandlers.NewOpenAPIHandler(fleetManagerJSONBytes).Get)
}

func init() {
	jsonBytes, err := yaml.YAMLToJSON(fleetManagerYAML)
	if err != nil {
		panic(err)
	}
	fleetManagerJSONBytes = jsonBytes
}
