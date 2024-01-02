package environments

import (
	"github.com/goava/di"
)

// ServiceProvider ...
type ServiceProvider interface {
	Providers() di.Option
}

// Func ...
func Func(f func() di.Option) func() ServiceProvider {
	return func() ServiceProvider {
		return providerFunc{apply: f}
	}
}

type providerFunc struct {
	apply func() di.Option
}

// Providers ...
func (s providerFunc) Providers() di.Option {
	return s.apply()
}
