package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Terraform struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec TerraformSpec `json:"spec,omitempty"`
	Status TerraformStatus `json:"status,omitempty"`
}

type TerraformSpec struct {
	FleetshardSync *FleetshardSyncSpec `json:"fleetshardSync,omitempty"`
	AcsOperator *AcsOperatorSpec `json:"acsOperator,omitempty"`
	Observability *ObservabilitySpec `json:"observability,omitempty"`
}

type TerraformStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}

type FleetshardSyncSpec struct {
	OcmToken string `json:"ocmToken,omitempty"`
	FleetManagerEndpoint string `json:"fleetManagerEndpoint,omitempty"`
	ClusterId string `json:"clusterId,omitempty"`
	RedHatSSO *RedHatSSOSpec `json:"redHatSSO,omitempty"`
}

type RedHatSSOSpec struct {
	ClientId string `json:"clientId,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
}

type AcsOperatorSpec struct {
	Enabled bool `json:"enabled,omitempty"`
	StartingCSV string `json:"startingCSV,omitempty"`
}

type ObservabilitySpec struct {
	Enabled bool `json:"enabled,omitempty"`
	Github *GithubSpec `json:"github,omitempty"`
	Observatorium *ObservatoriumSpec `json:"observatorium,omitempty"`
}

type GithubSpec struct {
	AccessToken string `json:"accessToken,omitempty"`
	Repository string `json:"repository,omitempty"`
}

type ObservatoriumSpec struct {
	Gateway string `json:"gateway,omitempty"`
	MetricsClientId string `json:"metricsClientId,omitempty"`
	MetricsSecret string `json:"metricsSecret,omitempty"`
}
