package declarativeconfigclone

import (
	"bytes"
	"io"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"gopkg.in/yaml.v3"
)

// TODO: Remove this file once declarative configuration in stackrox/stackrox fully supports notifiers

func supportedConfigurationTypes() string {
	return strings.Join([]string{
		declarativeconfig.AuthProviderConfiguration,
		declarativeconfig.AccessScopeConfiguration,
		declarativeconfig.PermissionSetConfiguration,
		declarativeconfig.RoleConfiguration,
		declarativeconfig.NotifierConfiguration,
	}, ",")
}

// ConfigurationFromRawBytes takes in a list of raw bytes, i.e. file contents, and returns the unmarshalled
// configurations.
// It will return an error if:
//   - the raw bytes are in invalid format, i.e. not YAML format.
//   - the YAML cannot be unmarshalled into valid configuration type.
func ConfigurationFromRawBytes(rawConfigurations ...[]byte) ([]declarativeconfig.Configuration, error) {
	var configurations []declarativeconfig.Configuration
	for _, rawConfiguration := range rawConfigurations {
		configs, err := parseToConfiguration(rawConfiguration)
		if err != nil {
			return nil, err
		}
		configurations = append(configurations, configs...)
	}

	return configurations, nil
}

func fromUnstructuredConfigs(unstructuredConfigs []interface{}) ([]declarativeconfig.Configuration, error) {
	configurations := make([]declarativeconfig.Configuration, 0, len(unstructuredConfigs))
	for _, unstructuredConfig := range unstructuredConfigs {
		config, err := fromUnstructured(unstructuredConfig)
		if err != nil {
			return nil, err
		}
		configurations = append(configurations, config)
	}
	return configurations, nil
}

func fromUnstructured(unstructured interface{}) (declarativeconfig.Configuration, error) {
	rawConfiguration, err := yaml.Marshal(unstructured)
	if err != nil {
		return nil, errors.Wrap(err, "marshalling unstructured configuration")
	}

	configs := []declarativeconfig.Configuration{
		&declarativeconfig.AuthProvider{},
		&declarativeconfig.AccessScope{},
		&declarativeconfig.PermissionSet{},
		&declarativeconfig.Role{},
		&declarativeconfig.Notifier{},
	}
	for _, c := range configs {
		err := decodeYAMLToConfiguration(rawConfiguration, c)
		if err == nil {
			return c, nil
		}
		if errors.Is(err, errox.InvalidArgs) {
			return nil, err
		}
	}
	return nil, errox.InvalidArgs.Newf("could not unmarshal configuration into any of the supported types [%s]",
		supportedConfigurationTypes())
}

func decodeYAMLToConfiguration(rawYAML []byte, configuration declarativeconfig.Configuration) error {
	dec := yaml.NewDecoder(bytes.NewReader(rawYAML))
	dec.KnownFields(true)
	if err := dec.Decode(configuration); err != nil {
		return err
	}
	return nil
}

func parseToConfiguration(contents []byte) ([]declarativeconfig.Configuration, error) {
	dec := yaml.NewDecoder(bytes.NewReader(contents))
	var unstructuredObjs []interface{}
	for {
		var obj interface{}
		err := dec.Decode(&obj)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "decoding YAML file contents")
		}
		unstructuredObjs = append(unstructuredObjs, obj)
	}

	var configurations []declarativeconfig.Configuration
	for _, unstructured := range unstructuredObjs {
		// Special case: a list of objects.
		listOfObj, ok := unstructured.([]interface{})
		if ok {
			configs, err := fromUnstructuredConfigs(listOfObj)
			if err != nil {
				return nil, err
			}
			configurations = append(configurations, configs...)
			continue
		}

		config, err := fromUnstructured(unstructured)
		if err != nil {
			return nil, err
		}
		configurations = append(configurations, config)
	}

	return configurations, nil
}
