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

// ArgoCdApplicationPluginParameter struct for ArgoCdApplicationPluginParameter
type ArgoCdApplicationPluginParameter struct {
	Name   string            `json:"name,omitempty"`
	String string            `json:"string,omitempty"`
	Array  []string          `json:"array,omitempty"`
	Map    map[string]string `json:"map,omitempty"`
}
