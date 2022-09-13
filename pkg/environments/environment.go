package environments

import (
	"context"
	goerrors "errors"
	"flag"
	"fmt"
	"os"

	"github.com/goava/di"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/defaults"

	sentryGo "github.com/getsentry/sentry-go"
	"github.com/spf13/pflag"
	glog "github.com/stackrox/acs-fleet-manager/pkg/logging"
)

// TestingEnv ...
const (
	TestingEnv     string = "testing"
	DevelopmentEnv string = "development"
	ProductionEnv  string = "production"
	StageEnv       string = "stage"
	IntegrationEnv string = "integration"

	EnvironmentStringKey string = "OCM_ENV"
	EnvironmentDefault          = DevelopmentEnv
)

// Env is a modular application built with dependency injection and manages
// applying lifecycle events of types injected  into the application.
//
// An Env uses two dependency injection containers.  The first one
// it constructs is the ConfigContainer
type Env struct {
	Name             string
	ConfigContainer  *di.Container
	ServiceContainer *di.Container
}

// New creates and initializes an Env with the provided name and injection
// options.  After initialization, types in the Env.ConfigContainer can
// be resolved using dependency injection.
func New(name string, options ...di.Option) (env *Env, err error) {
	env = &Env{
		Name: name,
	}

	env.ConfigContainer, err = di.New(append(options, di.ProvideValue(env))...)
	if err != nil {
		return nil, fmt.Errorf("initializing dependency injection: %w", err)
	}

	return env, nil
}

// GetEnvironmentStrFromEnv ...
func GetEnvironmentStrFromEnv() string {
	envStr, specified := os.LookupEnv(EnvironmentStringKey)
	if !specified || envStr == "" {
		glog.Infof("Environment variable %q not specified, using default %q", EnvironmentStringKey, EnvironmentDefault)
		envStr = EnvironmentDefault
	}
	return envStr
}

// AddFlags is used allow command command line flags to modify the values of types in
// the Env.ConfigContainer.  All types in Env.ConfigContainer that implement the ConfigModule
// interface invoked with ConfigModule.AddFlags.
//
// Then flag defaults will set by getting defaults for the Env.Name by looking up a EnvLoader
// tagged with that name in the  Env.ConfigContainer.
func (env *Env) AddFlags(flags *pflag.FlagSet) error {

	flags.AddGoFlagSet(flag.CommandLine)

	var namedEnv EnvLoader
	err := env.ConfigContainer.Resolve(&namedEnv, di.Tags{"env": env.Name})
	if err != nil {
		return errors.Errorf("unsupported environment %q", env.Name)
	}

	modules := []ConfigModule{}
	if err := env.ConfigContainer.Resolve(&modules); err != nil && !goerrors.Is(err, di.ErrTypeNotExists) {
		return fmt.Errorf("adding flags: %w", err)
	}
	for i := range modules {
		modules[i].AddFlags(flags)
	}

	return setConfigDefaults(flags, namedEnv.Defaults())
}

// CreateServices loads, creates, and validates the applications services.
//
// This should be called after the environment has been configured appropriately though AddFlags and parsing,
// done elsewhere.
//
// The function will look for many inject types in dependency injection container and called them in this order:
// 1) All ConfigModule.ReadFiles functions - to load file system based configuration.
// 2) The EnvLoader.ModifyConfiguration function - to allow named environment to apply configuration changes  (TODO: replace with a BeforeCreateServicesHook??)
// 3) All BeforeCreateServicesHook.Func functions - configuration hooks
// 4) All ServiceProvider.Providers  functions - to collect al dependency injection objects use to construct the Env.ServiceContainer
// 5) The Env.ServiceContainer is created
// 6) All ServiceValidator.Validate functions - used to validate the configuration or service types (TODO: replace with a AfterCreateServicesHook??)
// 7) All AfterCreateServicesHook.Func functions - a hook that is called after the service container is created.
func (env *Env) CreateServices() error {

	glog.Infof("Initializing %s environment", env.Name)

	// Read in all config files
	modules := []ConfigModule{}
	if err := env.ConfigContainer.Resolve(&modules); err != nil && !goerrors.Is(err, di.ErrTypeNotExists) {
		return fmt.Errorf("creating services: %w", err)
	}
	for i := range modules {
		err := modules[i].ReadFiles()
		if err != nil {
			err = errors.Errorf("unable to read configuration files: %s", err)
			glog.Error(err)
			sentryGo.CaptureException(err)
			return err
		}
	}

	// Allow named env to customize the configuration.
	var namedEnv EnvLoader
	err := env.ConfigContainer.Resolve(&namedEnv, di.Tags{"env": env.Name})
	if err != nil {
		return errors.Errorf("unsupported environment %q", env.Name)
	}
	err = namedEnv.ModifyConfiguration(env)
	if err != nil {
		return fmt.Errorf("modifying configuration: %w", err)
	}
	glog.Info("Active defaults for new Central tenants:")
	tryPrettyPrintDefaults(defaults.Central, "CentralDefaults")
	tryPrettyPrintDefaults(defaults.Scanner, "ScannerDefaults")

	type injections struct {
		di.Inject
		ServiceInjections         []ServiceProvider
		BeforeCreateServicesHooks []BeforeCreateServicesHook `optional:"true"`
		AfterCreateServicesHooks  []AfterCreateServicesHook  `optional:"true"`
	}
	in := injections{}
	if err := env.ConfigContainer.Resolve(&in); err != nil {
		return fmt.Errorf("creating services: %w", err)
	}

	// Right before we create the services, run the any before create service hooks
	// to allow for post config file loading customization.
	for _, hook := range in.BeforeCreateServicesHooks {
		env.MustInvoke(hook.Func)
	}

	// Lets build a list of providers that will be used to create the service container.
	var serviceProviders []di.Option
	for _, s := range in.ServiceInjections {
		serviceProviders = append(serviceProviders, s.Providers())
	}
	env.ServiceContainer, err = di.New(serviceProviders...)
	if err != nil {
		return fmt.Errorf("creating services: %w", err)
	}

	// add the parent so the ServiceContainer can automatically resolve
	// types in the ConfigContainer.
	err = env.ServiceContainer.AddParent(env.ConfigContainer)
	if err != nil {
		return fmt.Errorf("creating services: %w", err)
	}

	var validators []ServiceValidator
	err = env.ServiceContainer.Resolve(&validators)
	if err != nil {
		if !errors.Is(err, di.ErrTypeNotExists) {
			return fmt.Errorf("creating services: %w", err)
		}
	} else {
		for _, validator := range validators {
			if err := validator.Validate(); err != nil {
				return fmt.Errorf("creating services: %w", err)
			}
		}
	}

	for _, hook := range in.AfterCreateServicesHooks {
		env.MustInvoke(hook.Func)
	}

	return nil
}

// MustInvoke ...
func (env *Env) MustInvoke(invocation di.Invocation, options ...di.InvokeOption) {
	container := env.ServiceContainer
	containerName := "service container"
	if container == nil {
		container = env.ConfigContainer
		containerName = "config container"
	}
	if err := container.Invoke(invocation, options...); err != nil {
		glog.Fatalf("%s di failure: %v", containerName, err)
	}
}

// MustResolve ...
func (env *Env) MustResolve(ptr di.Pointer, options ...di.ResolveOption) {
	container := env.ServiceContainer
	containerName := "service container"
	if container == nil {
		container = env.ConfigContainer
		containerName = "config container"
	}
	if err := container.Resolve(ptr, options...); err != nil {
		glog.Fatalf("%s di failure: %v", containerName, err)
	}
}

// MustResolveAll ...
func (env *Env) MustResolveAll(ptrs ...di.Pointer) {
	container := env.ServiceContainer
	containerName := "service container"
	if container == nil {
		container = env.ConfigContainer
		containerName = "config container"
	}
	for _, ptr := range ptrs {
		if err := container.Resolve(ptr); err != nil {
			glog.Fatalf("%s di failure: %v", containerName, err)
		}
	}
}

// Run starts the Env waits for the context to be canceled and then stops the Env.
func (env *Env) Run(ctx context.Context) {
	env.Start()
	<-ctx.Done()
	env.Stop()
}

// Start calls all the BootService.Start functions found in the container.
func (env *Env) Start() {
	env.MustInvoke(func(services []BootService) {
		for i := range services {
			services[i].Start()
		}
	})
}

// Stop calls all the BootService.Stop functions found in the dependency injection containers.
func (env *Env) Stop() {
	glog.Infof("Stopping all boot services.")
	env.MustInvoke(func(services []BootService) {
		for i := range services {
			i = len(services) - 1 - i // to stop in reverse order
			services[i].Stop()
		}
	})
}

// Cleanup calls all the cleanup functions registered with the dependency injection  containers.
func (env *Env) Cleanup() {
	if env.ServiceContainer != nil {
		env.ServiceContainer.Cleanup()
	}
	env.ConfigContainer.Cleanup()
}

func setConfigDefaults(flags *pflag.FlagSet, defaults map[string]string) error {
	for name, value := range defaults {
		err := flags.Set(name, value)
		if err != nil {
			glog.Errorf("Error setting flag %s: %v", name, err)
			return fmt.Errorf("setting flag: %w", err)
		}
	}
	return nil
}

func tryPrettyPrintDefaults(obj interface{}, label string) {
	prettyPrintedDefaults, err := defaults.PrettyPrintDefaults(obj, label)
	if err != nil {
		glog.Errorf("Failed to pretty-print %s: %v", label, err)
		glog.Errorf("%s: %+v", label, obj)
	} else {
		for _, line := range prettyPrintedDefaults {
			glog.Info("  " + line)
		}
	}
}
