package gitops

import (
	"os"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/yaml"
)

// ConfigProvider is the interface for GitOps configuration providers.
type ConfigProvider interface {
	// Get returns the GitOps configuration.
	Get() (Config, error)
}

// -- defaultConfigProvider --

// NewDefaultConfigProvider returns a new default ConfigProvider.
func NewDefaultConfigProvider() ConfigProvider {
	return NewFallbackToLastWorkingConfigProvider(
		NewProviderWithMetrics(
			NewEmptyConfigProvider(),
			// TODO: Use a fileConfigProvider instead of an emptyConfigProvider
		),
	)
}

// -- fileConfigProvider --

// fileConfigProvider is a ConfigProvider that reads GitOps configuration from a file.
type fileConfigProvider struct {
	path string
}

// NewFileConfigProvider returns a new fileConfigProvider
func NewFileConfigProvider(path string) ConfigProvider {
	return &fileConfigProvider{path: path}
}

// Get implements ConfigProvider.Get
func (p *fileConfigProvider) Get() (Config, error) {
	fileBytes, err := os.ReadFile(p.path)
	if err != nil {
		return Config{}, errors.Wrap(err, "failed to read GitOps configuration file")
	}
	var config Config
	if err := yaml.Unmarshal(fileBytes, &config); err != nil {
		return Config{}, errors.Wrap(err, "failed to unmarshal GitOps configuration")
	}
	return config, nil
}

// -- staticConfigProvider --

// staticConfigProvider is a ConfigProvider that returns a static configuration.
type staticConfigProvider struct {
	config Config
}

// NewStaticConfigProvider returns a new staticConfigProvider
func NewStaticConfigProvider(config Config) ConfigProvider {
	return &staticConfigProvider{config: config}
}

// Get implements ConfigProvider.Get
func (p *staticConfigProvider) Get() (Config, error) {
	return p.config, nil
}

// -- emptyConfigProvider --

// emptyConfigProvider is a ConfigProvider that returns an empty configuration.
type emptyConfigProvider struct{}

// NewEmptyConfigProvider returns a new emptyConfigProvider
func NewEmptyConfigProvider() ConfigProvider {
	return &staticConfigProvider{
		config: Config{},
	}
}

// -- fallbackToLastWorkingConfigProvider --

// fallbackToLastWorkingConfigProvider is a ConfigProvider that returns the configuration from the primary
// ConfigProvider, but if the primary ConfigProvider fails, it returns the last working configuration.
type fallbackToLastWorkingConfigProvider struct {
	primary           ConfigProvider
	lastWorkingConfig atomic.Pointer[Config]
}

// NewFallbackToLastWorkingConfigProvider returns a new fallbackToLastWorkingConfigProvider
func NewFallbackToLastWorkingConfigProvider(primary ConfigProvider) ConfigProvider {
	return &fallbackToLastWorkingConfigProvider{primary: primary}
}

// Get implements ConfigProvider.Get
func (p *fallbackToLastWorkingConfigProvider) Get() (Config, error) {
	config, err := p.primary.Get()
	if err != nil {
		return p.getLastWorkingConfig(err)
	}
	p.lastWorkingConfig.Store(&config)
	return config, nil
}

// getLastWorkingConfig returns the last working configuration, or an error if there is no last working configuration.
func (p *fallbackToLastWorkingConfigProvider) getLastWorkingConfig(err error) (Config, error) {
	lastWorkingConfig := p.lastWorkingConfig.Load()
	if lastWorkingConfig == nil {
		return Config{}, errors.Wrap(err, "failed to get GitOps configuration")
	}
	return *lastWorkingConfig, nil
}

// -- providerWithMetrics --

// providerWithMetrics is a ConfigProvider that wraps another ConfigProvider and emits metrics.
type providerWithMetrics struct {
	delegate ConfigProvider
}

// NewProviderWithMetrics returns a new providerWithMetrics
func NewProviderWithMetrics(delegate ConfigProvider) ConfigProvider {
	return &providerWithMetrics{delegate: delegate}
}

var (
	errorCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dinosaur_gitops_config_provider_error_count",
		Help: "Number of errors encountered by the GitOps configuration provider.",
	}, []string{"error"})
)

// Get implements ConfigProvider.Get
func (p *providerWithMetrics) Get() (Config, error) {
	config, err := p.delegate.Get()
	if err != nil {
		errorCounter.WithLabelValues(err.Error()).Inc()
	}
	return config, errors.Wrap(err, "failed to get delegate GitOps configuration")
}

func init() {
	prometheus.MustRegister(errorCounter)
}
