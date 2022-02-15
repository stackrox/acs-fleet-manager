package shared

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"github.com/ghodss/yaml"
)

func LoadOpenAPISpec(assetFunc func(name string) ([]byte, error), asset string) (data []byte, err error) {
	data, err = assetFunc(asset)
	if err != nil {
		err = errors.GeneralError(
			"can't load OpenAPI specification from asset '%s'",
			asset,
		)
		return
	}
	data, err = yaml.YAMLToJSON(data)
	if err != nil {
		err = errors.GeneralError(
			"can't convert OpenAPI specification loaded from asset '%s' from YAML to JSON",
			asset,
		)
		return
	}
	return
}
