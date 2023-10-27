package gitops

import (
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/features"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
)

// ConfigProvider is the interface for GitOps configuration providers.
type ConfigProvider interface {
	// Get returns the GitOps configuration.
	Get() (Config, error)
}

type validationFn func(config Config) error

type provider struct {
	reader            Reader
	lastWorkingConfig atomic.Pointer[Config]
	validationFn      validationFn
}

var (
	configProviderInstance ConfigProvider
	onceConfigProvider     sync.Once
)

// GetConfigProvider returns a new ConfigProvider.
func GetConfigProvider() ConfigProvider {
	onceConfigProvider.Do(func() {
		var reader Reader
		if features.GitOpsCentrals.Enabled() {
			reader = NewFileReader(configPath)
		} else {
			reader = NewEmptyReader()
		}

		configProviderInstance = &provider{
			reader:            reader,
			lastWorkingConfig: atomic.Pointer[Config]{},
			validationFn: func(config Config) error {
				return ValidateConfig(config).ToAggregate()
			},
		}
	})
	return configProviderInstance
}

// Get implements ConfigProvider.Get
func (p *provider) Get() (Config, error) {
	// Load the config from the reader
	cfg, err := p.reader.Read()
	if err != nil {
		p.increaseErrorCount()
		return p.tryGetLastWorkingConfig(errors.Wrap(err, "failed to read GitOps configuration"))
	}
	lastWorkingConfig := p.lastWorkingConfig.Load()
	if lastWorkingConfig != nil {
		if reflect.DeepEqual(cfg, *lastWorkingConfig) {
			return cfg, nil
		}
	}
	// Validate the config
	if err := p.validationFn(cfg); err != nil {
		p.increaseErrorCount()
		return p.tryGetLastWorkingConfig(errors.Wrap(err, "failed to validate GitOps configuration"))
	}
	// Store the config as the last working config
	p.lastWorkingConfig.Store(&cfg)
	return cfg, nil
}

func (p *provider) increaseErrorCount() {
	metrics.GitopsConfigProviderErrorCounter.WithLabelValues().Inc()
}

func (p *provider) tryGetLastWorkingConfig(err error) (Config, error) {
	lastWorkingConfig := p.lastWorkingConfig.Load()
	if lastWorkingConfig == nil {
		return Config{}, errors.Wrap(err, "no last working gitops config available")
	}
	glog.Warningf("Failed to get GitOps configuration. Using last working config: %s", err)
	return *lastWorkingConfig, nil
}
