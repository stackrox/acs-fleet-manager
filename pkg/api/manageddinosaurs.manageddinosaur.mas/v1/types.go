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

type VersionsSpec struct {
	Dinosaur         string `json:"dinosaur"`
	DinosaurOperator string `json:"dinosaurOperator"`
}

type ManagedDinosaurStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
	Versions   VersionsSpec       `json:"versions"`
}

// Spec
type TlsSpec struct {
	Cert string `json:"cert"`
	Key  string `json:"key"`
}

type EndpointSpec struct {
	Host string   `json:"host"`
	Tls  *TlsSpec `json:"tls,omitempty"`
}

type AuthSpec struct {
	ClientSecret string `json:"clientSecret,omitempty"`
	ClientId     string `json:"clientId,omitempty"`
	OwnerUserId  string `json:"ownerUserId,omitempty"`
	OwnerOrgId   string `json:"ownerOrgId,omitempty"`
}

type ManagedDinosaurSpec struct {
	Auth     AuthSpec     `json:"auth"`
	Endpoint EndpointSpec `json:"endpoint"`
	Versions VersionsSpec `json:"versions"`
	Deleted  bool         `json:"deleted"`
	Owners   []string     `json:"owners"`
}

type ManagedDinosaur struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Id     string                `json:"id,omitempty"`
	Spec   ManagedDinosaurSpec   `json:"spec,omitempty"`
	Status ManagedDinosaurStatus `json:"status,omitempty"`
}
