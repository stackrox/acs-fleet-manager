# Adding Flags to Fleet Manager
This document outlines best practices and guides on how to add new flags to the Fleet Manager's serve command.

- [Naming Conventions](#naming-conventions)
    - [Feature Flags](#feature-flags)
- [Adding a New Flag](#adding-a-new-flag)
    - [Adding to an Existing Config File](#adding-to-an-existing-config-file)
    - [Adding a New Config File](#adding-a-new-config-file)
    - [Verify Addition of New Flags](#verify-addition-of-new-flags)
- [Adding Flags to the Service Template](#adding-flags-to-the-service-template)
- [Documenting Feature Flags](#documenting-feature-flags)

## Naming conventions
In general, flag names should be all lower case using the kebab case style: `flag-name` (e.g. `public-host-url`).

### Feature Flags
Feature flags are flags that enables/disables features of the service.

For any feature flags that expects a boolean value, it should have a `enable-` prefix followed by the feature description: `enable-<feature>` (e.g. `enable-deny-list`). If the flag is set to `true`, that should indicate that the feature is enabled. If the flag is set to `false`, that should indicate that the feature is disabled.

Feature flags can also expect a string value. In this case, it should follow the general flag naming convention (e.g. `dataplane-cluster-scaling-type`).

Any sub-configuration flags that needs to be specified if a feature is enabled should also follow the general flag naming convention.

## Adding a New Flag
Flags are defined within a configuration file located in the *pkg/config/* directory.

Flags should assign its value to a property within a config struct or a variable within the configuration file.

> NOTE: Fleet Manager uses [spf13/pflag](https://github.com/spf13/pflag) for flags management.

### Adding to an Existing Config File
1. Select a configuration file in the *pkg/config/* directory that is relevant to the new feature that is being added.
2. Add the flag to the `AddFlags()` function within that configuration file.
3. If a new feature flag is being added, any sub-configuration flags for this feature may also be defined here.

Example:
```go
func (c *Config) AddFlags(fs *pflag.FlagSet) {
    ...
    fs.StringVar(&c.NewFlag, "new-flag", NewFlag, "A new flag")
    fs.BoolVar(&c.EnableNewFeature, "enable-new-feature", EnableNewFeature, "A new feature flag (default: true)")
	fs.IntVar(&c.NewFeatureSubConfig1, "new-feature-sub-config-1", NewFeatureSubConfig1, "A configuration that needs to be specified by the user if the new feature is enabled (default: 1)")
	fs.StringVar(&c.NewFeatureSubConfig2, "new-feature-sub-config-2", NewFeatureSubConfig2, "A configuration that needs to be specified by the user if the new feature is enabled")
}
```

### Adding a New Config File
If the new configuration flag doesn't fit in any of the existing config file, a new one should be created.

1. Create a new config file with a filename format of `<feature>.go` under the *internal/<resource>/config/* directory (e.g. `internal/dinosaur/pkg/config/`).  (See [dinosaur.go](../internal/dinosaur/pkg/config/dinosaur.go) as an example)
2. Define any new flags in the `AddFlags()` function.
3. New config file has to implement `ConfigModule` interface and to be added into the `CoreConfigProviders()` function inside the [core providers file](../pkg/providers/core.go) or any more appropriate internal folder, e.g. [dinosaur providers](../internal/dinosaur/providers.go) (see `ConfigProviders()` function)

### Verify Addition of New Flags
Flags defined in configuration files in *pkg/config/* are added to the Fleet Manager **serve** command.

To verify that a new flag has been successfully added, run the following commands.

```bash
    make binary
    ./fleet-manager serve -h
```

This will list all of the available flags that can be specified with the **serve** command. Any new flags should be listed here.

Once values are set, these configurations will be available in the overall Application Config so that these values can be accessed within the code.

Example:
```go
    ...
    env := env
    if env.Config.Sentry.Enabled {
        ...
    }
    ...
```

## Adding Flags to the Service Template
Once flags have been defined, they should be added as parameters to the Fleet Manager's service template. The parameters should then be used as the flag value in the serve command ran within the service container of the Fleet Manager deployment .

See the [service template](../templates/service-template.yml) for more information.

Any new template parameters should be added as parameters to the `deploy` make target in the [Makefile](../Makefile). These additional parameters should be documented in the optional parameters section of the [Running the Service on OpenShift](../README.md#deploy-the-service-using-templates) guide

## Documenting Feature Flags
Any new feature flags should be documented [here](./feature-flags.md). Any sub-configuration that can be set if this feature is enabled should also be documented. It should state if the sub-configuration is required/optional as well as their default values if it has one.

Any flags that should no longer be used but cannot be removed at the current time should be marked as `deprecated`. A note should be added as to the reason why it is deprecated as well as stating what alternatives can be used if there is one.

Any flags that will be removed at a later point in time should have its lifecycle information documented.

For example:

**Feature**
- **enable-feature-1** _(deprecated)_: \<feature flag description>
    > This flag has been deprecated because of _x_, please use \<alternative-flag> instead.
    - `sub-configuration-flag`: \<description of flag>
- **enable-feature-2**: \<feature flag description>
    > This is a temporary feature and will be removed on \<date>
- **enable-feature-3**: \<feature flag description>
    - `sub-configuration-flag-1` _(deprecated)_: \<description of flag>
        > This flag has been deprecated because of _x_, please use sub-configuration-flag-2 instead.
    - `sub-configuration-flag-2`: \<description of flag>
