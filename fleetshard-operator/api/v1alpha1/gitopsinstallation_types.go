package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GitopsInstallation is the Schema for the gitopsinstallations API
type GitopsInstallation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitopsInstallationSpec   `json:"spec,omitempty"`
	Status GitopsInstallationStatus `json:"status,omitempty"`
}

// GitopsInstallationSpec defines the desired state of GitopsInstallation
type GitopsInstallationSpec struct {
	// ClusterName used for figuring out the boostrap profile.
	ClusterName string `json:"clusterName,omitempty"`
	// BootstrapAppTargetRevision allows to explicitly set the bootstrap app target revision.
	BootstrapAppTargetRevision string `json:"bootstrapAppTargetRevision,omitempty"`
}

// GitopsInstallationStatus defines the observed state of GitopsInstallation
type GitopsInstallationStatus struct{}

type GitopsInstallationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitopsInstallation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GitopsInstallation{}, &GitopsInstallationList{})
}
