/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VersionsSpec ...
type VersionsSpec struct {
	Dinosaur         string `json:"dinosaur"`
	DinosaurOperator string `json:"dinosaurOperator"`
}

// ManagedDinosaurStatus ...
type ManagedDinosaurStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
	Versions   VersionsSpec       `json:"versions"`
}

// TLSSpec Spec
type TLSSpec struct {
	Cert string `json:"cert"`
	Key  string `json:"key"`
}

// EndpointSpec ...
type EndpointSpec struct {
	Host string   `json:"host"`
	TLS  *TLSSpec `json:"tls,omitempty"`
}

// AuthSpec ...
type AuthSpec struct {
	ClientSecret string `json:"clientSecret,omitempty"`
	ClientID     string `json:"clientId,omitempty"`
	OwnerUserID  string `json:"ownerUserId,omitempty"`
	OwnerOrgID   string `json:"ownerOrgId,omitempty"`
	Issuer       string `json:"issuer,omitempty"`
}

// ManagedDinosaurSpec ...
type ManagedDinosaurSpec struct {
	Auth     AuthSpec     `json:"auth"`
	Endpoint EndpointSpec `json:"endpoint"`
	Versions VersionsSpec `json:"versions"`
	Deleted  bool         `json:"deleted"`
	Owners   []string     `json:"owners"`
}

// ManagedDinosaur ...
type ManagedDinosaur struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	ID            string                `json:"id,omitempty"`
	Spec          ManagedDinosaurSpec   `json:"spec,omitempty"`
	Status        ManagedDinosaurStatus `json:"status,omitempty"`
	RequestStatus string                `json:"requestStatus,omitempty"`
}
