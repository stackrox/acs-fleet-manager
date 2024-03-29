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

// List struct for List
type List struct {
	Kind  string `json:"kind"`
	Page  int32  `json:"page"`
	Size  int32  `json:"size"`
	Total int32  `json:"total"`
}
