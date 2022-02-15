/*
 * Dinosaur Service Fleet Manager
 *
 * Dinosaur Service Fleet Manager is a Rest API to manage Dinosaur instances.
 *
 * API version: 1.2.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

// CloudProviderList struct for CloudProviderList
type CloudProviderList struct {
	Kind  string          `json:"kind"`
	Page  int32           `json:"page"`
	Size  int32           `json:"size"`
	Total int32           `json:"total"`
	Items []CloudProvider `json:"items"`
}
