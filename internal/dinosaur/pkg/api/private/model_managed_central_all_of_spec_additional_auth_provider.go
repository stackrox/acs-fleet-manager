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

// ManagedCentralAllOfSpecAdditionalAuthProvider struct for ManagedCentralAllOfSpecAdditionalAuthProvider
type ManagedCentralAllOfSpecAdditionalAuthProvider struct {
	Name               string                                                            `json:"name,omitempty"`
	MinimumRoleName    string                                                            `json:"minimumRoleName,omitempty"`
	Groups             []ManagedCentralAllOfSpecAdditionalAuthProviderGroups             `json:"groups,omitempty"`
	RequiredAttributes []ManagedCentralAllOfSpecAdditionalAuthProviderRequiredAttributes `json:"requiredAttributes,omitempty"`
	ClaimMappings      []ManagedCentralAllOfSpecAdditionalAuthProviderRequiredAttributes `json:"claimMappings,omitempty"`
	Oidc               ManagedCentralAllOfSpecAdditionalAuthProviderOidc                 `json:"oidc,omitempty"`
}
