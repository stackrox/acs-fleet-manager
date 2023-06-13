// Package features helps enable or disable features.
package features

import (
	"fmt"
	"strings"
)

// A FeatureFlag is a product behavior that can be enabled or disabled using an environment variable.
type FeatureFlag interface {
	Name() string
	EnvVar() string
	Enabled() bool
	Default() bool
}

var (
	// Flags contains all defined FeatureFlags by name.
	Flags = make(map[string]FeatureFlag)
)

func registerFeature(name, envVar string, defaultValue bool) FeatureFlag {
	if !strings.HasPrefix(envVar, "RHACS_") {
		panic(fmt.Sprintf("invalid env var: %s, must start with RHACS_", envVar))
	}
	f := &feature{
		name:         name,
		envVar:       envVar,
		defaultValue: defaultValue,
	}
	Flags[f.Name()] = f
	return f
}
