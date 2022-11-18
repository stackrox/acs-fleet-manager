/*
 * Red Hat Advanced Cluster Security Service Fleet Manager
 *
 * Red Hat Advanced Cluster Security (RHACS) Service Fleet Manager is a Rest API to manage instances of ACS components.
 *
 * API version: 1.2.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech). DO NOT EDIT.
package public

// CentralRequestPayload Schema for the request body sent to /centrals POST
type CentralRequestPayload struct {
	// The cloud provider where the Central component will be created in
	CloudProvider string `json:"cloud_provider,omitempty"`
	// The cloud account ID that is linked to the ACS instance
	CloudAccountId string `json:"cloud_account_id,omitempty"`
	// Set this to true to configure the Central component to be multiAZ
	MultiAz bool `json:"multi_az,omitempty"`
	// The name of the Central component. It must consist of lower-case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character, and can not be longer than 32 characters.
	Name string `json:"name"`
	// The region where the Central component cluster will be created in
	Region  string      `json:"region,omitempty"`
	Central CentralSpec `json:"central,omitempty"`
	Scanner ScannerSpec `json:"scanner,omitempty"`
}
