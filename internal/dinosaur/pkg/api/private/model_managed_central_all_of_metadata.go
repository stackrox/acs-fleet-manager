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

// ManagedCentralAllOfMetadata struct for ManagedCentralAllOfMetadata
type ManagedCentralAllOfMetadata struct {
	Name              string                                 `json:"name,omitempty"`
	Namespace         string                                 `json:"namespace,omitempty"`
	Annotations       ManagedCentralAllOfMetadataAnnotations `json:"annotations,omitempty"`
	DeletionTimestamp string                                 `json:"deletionTimestamp,omitempty"`
}
