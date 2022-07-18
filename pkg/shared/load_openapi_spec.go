package shared

import (
	"github.com/ghodss/yaml"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
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
