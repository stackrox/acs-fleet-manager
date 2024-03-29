/*
 * Red Hat Advanced Cluster Security Service Fleet Manager Admin API
 *
 * Red Hat Advanced Cluster Security (RHACS) Service Fleet Manager Admin APIs that can be used by RHACS Managed Service Operations Team.
 *
 * API version: 0.0.3
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech). DO NOT EDIT.
package private

// CentralBillingChangeRequest struct for CentralBillingChangeRequest
type CentralBillingChangeRequest struct {
	Model          string `json:"model,omitempty"`
	CloudAccountId string `json:"cloud_account_id,omitempty"`
	CloudProvider  string `json:"cloud_provider,omitempty"`
	Product        string `json:"product,omitempty"`
}
