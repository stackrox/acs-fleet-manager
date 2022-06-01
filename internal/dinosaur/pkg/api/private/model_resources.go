/*
 * Red Hat Advanced Cluster Security Service Fleet Manager
 *
 * Red Hat Advanced Cluster Security (RHACS) Service Fleet Manager APIs that are used by internal services e.g fleetshard operators.
 *
 * API version: 1.4.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// Resources struct for Resources
type Resources struct {
	Requests ResourceReference `json:"requests,omitempty"`
	Limits   ResourceReference `json:"limits,omitempty"`
}
