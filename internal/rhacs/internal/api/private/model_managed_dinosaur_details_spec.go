/*
 * Dinosaur Service Fleet Manager
 *
 * Dinosaur Service Fleet Manager APIs that are used by internal services e.g fleetshard operators.
 *
 * API version: 1.4.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// ManagedDinosaurDetailsSpec struct for ManagedDinosaurDetailsSpec
type ManagedDinosaurDetailsSpec struct {
	Owners   []string                           `json:"owners,omitempty"`
	Endpoint ManagedDinosaurDetailsSpecEndpoint `json:"endpoint,omitempty"`
	Versions ManagedDinosaurVersions            `json:"versions,omitempty"`
	Deleted  bool                               `json:"deleted"`
}
