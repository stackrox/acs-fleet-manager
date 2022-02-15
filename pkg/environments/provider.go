package environments

import (
	"github.com/goava/di"
)

type ServiceProvider interface {
	Providers() di.Option
}

func Func(f func() di.Option) func() ServiceProvider {
	return func() ServiceProvider {
		return providerFunc{apply: f}
	}
}

type providerFunc struct {
	apply func() di.Option
}

func (s providerFunc) Providers() di.Option {
	return s.apply()
}
