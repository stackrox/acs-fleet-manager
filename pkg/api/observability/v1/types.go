package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObservabilityAuthType ...
type ObservabilityAuthType string

// AuthTypeDex ...
const (
	AuthTypeDex ObservabilityAuthType = "dex"
)

// DexConfig ...
type DexConfig struct {
	URL                       string `json:"url" yaml:"url"`
	CredentialSecretNamespace string `json:"credentialSecretNamespace" yaml:"credentialSecretNamespace"`
	CredentialSecretName      string `json:"credentialSecretName" yaml:"credentialSecretName"`
}

// GrafanaConfig ...
type GrafanaConfig struct {
	// If false, the operator will install default dashboards and ignore list
	Managed bool `json:"managed" yaml:"managed"`
}

// ObservatoriumConfig ...
type ObservatoriumConfig struct {
	// Observatorium Gateway API URL
	Gateway string `json:"gateway" yaml:"gateway"`
	// Observatorium tenant name
	Tenant string `json:"tenant" yaml:"tenant"`

	// Auth type. Currently only dex is supported
	AuthType ObservabilityAuthType `json:"authType,omitempty" yaml:"authType,omitempty"`

	// Dex configuration
	AuthDex *DexConfig `json:"dexConfig,omitempty" yaml:"dexConfig,omitempty"`
}

// AlertmanagerConfig ...
type AlertmanagerConfig struct {
	PagerDutySecretName           string `json:"pagerDutySecretName"`
	PagerDutySecretNamespace      string `json:"pagerDutySecretNamespace,omitempty"`
	DeadMansSnitchSecretName      string `json:"deadMansSnitchSecretName"`
	DeadMansSnitchSecretNamespace string `json:"deadMansSnitchSecretNamespace,omitempty"`
}

// ObservabilitySpec defines the desired state of Observability
type ObservabilitySpec struct {
	// Observatorium config
	Observatorium ObservatoriumConfig `json:"observatorium"`

	// Grafana config
	Grafana GrafanaConfig `json:"grafana"`

	// Alertmanager config
	Alertmanager AlertmanagerConfig `json:"alertmanager,omitempty"`

	// Selector for all namespaces that should be scraped
	DinosaurNamespaceSelector metav1.LabelSelector `json:"dinosaurNamespaceSelector"`

	// Selector for all canary pods that should be scraped
	CanaryPodSelector metav1.LabelSelector `json:"canaryPodSelector,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Observability is the Schema for the observabilities API
type Observability struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ObservabilitySpec `json:"spec,omitempty"`
}
