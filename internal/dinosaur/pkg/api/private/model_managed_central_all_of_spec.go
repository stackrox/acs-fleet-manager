/*
 * Red Hat Advanced Cluster Security Service Fleet Manager
 *
 * Red Hat Advanced Cluster Security (RHACS) Service Fleet Manager APIs that are used by internal services e.g fleetshard operators.
 *
 * API version: 1.4.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// ManagedCentralAllOfSpec struct for ManagedCentralAllOfSpec
type ManagedCentralAllOfSpec struct {
	Owners   []string                        `json:"owners,omitempty"`
	Endpoint ManagedCentralAllOfSpecEndpoint `json:"endpoint,omitempty"`
	Versions ManagedCentralVersions          `json:"versions,omitempty"`
	Central  ManagedCentralAllOfSpecCentral  `json:"central,omitempty"`
	Scanner  ManagedCentralAllOfSpecScanner  `json:"scanner,omitempty"`
}
