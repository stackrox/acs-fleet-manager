/*
 * Red Hat Advanced Cluster Security Service Fleet Manager
 *
 * Red Hat Advanced Cluster Security (RHACS) Service Fleet Manager APIs that are used by internal services e.g fleetshard operators.
 *
 * API version: 1.4.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// ManagedCentralAllOfSpecEndpoint struct for ManagedCentralAllOfSpecEndpoint
type ManagedCentralAllOfSpecEndpoint struct {
	Host string                             `json:"host,omitempty"`
	Tls  ManagedCentralAllOfSpecEndpointTls `json:"tls,omitempty"`
}
