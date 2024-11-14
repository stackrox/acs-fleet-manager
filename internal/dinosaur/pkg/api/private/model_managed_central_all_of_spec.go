/*
 * Red Hat Advanced Cluster Security Service Fleet Manager
 *
 * Red Hat Advanced Cluster Security (RHACS) Service Fleet Manager APIs that are used by internal services e.g fleetshard operators.
 *
 * API version: 1.4.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech). DO NOT EDIT.
package private

// ManagedCentralAllOfSpec struct for ManagedCentralAllOfSpec
type ManagedCentralAllOfSpec struct {
	InstanceType           string                                        `json:"instanceType,omitempty"`
	TenantResourcesValues  map[string]interface{}                        `json:"tenantResourcesValues,omitempty"`
	CentralCRYAML          string                                        `json:"centralCRYAML,omitempty"`
	Owners                 []string                                      `json:"owners,omitempty"`
	Auth                   ManagedCentralAllOfSpecAuth                   `json:"auth,omitempty"`
	AdditionalAuthProvider ManagedCentralAllOfSpecAdditionalAuthProvider `json:"additionalAuthProvider,omitempty"`
	UiEndpoint             ManagedCentralAllOfSpecUiEndpoint             `json:"uiEndpoint,omitempty"`
	DataEndpoint           ManagedCentralAllOfSpecDataEndpoint           `json:"dataEndpoint,omitempty"`
}
