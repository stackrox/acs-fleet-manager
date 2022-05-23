package config

import (
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"gopkg.in/yaml.v2"
	"testing"
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
dinosaur_instance_limit: 5
provider_type: standalone
supported_instance_type: "eval,standard"
cluster_dns: cluster.local
`)

	c := ManualCluster{}
	if err := yaml.Unmarshal(configFile, &c); err != nil {
		t.Fail()
	}
	if c.Status != api.ClusterReady {
		t.Fail()
	}
}
