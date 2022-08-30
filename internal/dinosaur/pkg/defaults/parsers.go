package defaults

import (
	"fmt"
	"reflect"

	env "github.com/caarlos0/env/v6"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	// QuantityType is a helper var that represents the `reflect.Type`` of `resource.Quantity`
	QuantityType = reflect.TypeOf(resource.Quantity{})

	// CustomParsers ...
	CustomParsers = map[reflect.Type]env.ParserFunc{
		QuantityType: QuantityParser,
	}
)

// QuantityParser is a basic parser for the resource.Quantity type that should be used with `env.ParseWithFuncs()`
func QuantityParser(v string) (interface{}, error) {
	qty, err := resource.ParseQuantity(v)
	if err != nil {
		return nil, fmt.Errorf("parsing quantity %q: %v", v, err)
	}

	return qty, nil
}
