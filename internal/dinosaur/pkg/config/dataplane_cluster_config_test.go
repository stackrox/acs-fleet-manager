package config

import (
	"testing"

	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"gopkg.in/yaml.v2"
)

// TODO(create-ticket): Add testing for config file marshaling or switch to a simpler format / parsing logic.
func Test_UnmarshalYAML(t *testing.T) {
	configFile := []byte(`
name: minikube  # Uncomment if using minikube
cluster_id: 1234567890abcdef1234567890abcdef
cloud_provider: standalone
region: standalone
schedulable: true
status: ready
central_instance_limit: 5
provider_type: standalone
supported_instance_type: "eval,standard"
cluster_dns: cluster.local
available_central_operator_versions:
  - version: "0.1.0"
    ready: true
    central_versions:
      - version: "0.1.0"
`)

	c := ManualCluster{}
	if err := yaml.Unmarshal(configFile, &c); err != nil {
		t.Fail()
	}

	if c.Status != api.ClusterReady {
		t.Fail()
	}

	if len(c.AvailableCentralOperatorVersions) < 1 {
		t.Fatal("Expected operator versions to not be empty")
	}

	want := "0.1.0"
	got := c.AvailableCentralOperatorVersions[0].Version
	if got != want {
		t.Fatalf("Expected first central operator version to be: %s, got: %s\n", want, got)
	}

	if len(c.AvailableCentralOperatorVersions[0].CentralVersions) < 1 {
		t.Fatal("Expected central versions to not be empty")
	}

	got = c.AvailableCentralOperatorVersions[0].CentralVersions[0].Version
	if got != want {
		t.Fatalf("Expected first central version to be: %s, got: %s\n", want, got)
	}
}
