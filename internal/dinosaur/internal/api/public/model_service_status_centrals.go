/*
 * Red Hat Advanced Cluster Security Service Fleet Manager
 *
 * Red Hat Advanced Cluster Security (RHACS) Service Fleet Manager is a Rest API to manage instances of ACS components.
 *
 * API version: 1.2.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

// ServiceStatusCentrals The RHACS resource api status
type ServiceStatusCentrals struct {
	// Indicates whether maximum service capacity has been reached
	MaxCapacityReached bool `json:"max_capacity_reached"`
}
