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

import (
	"time"
)

// ManagedCentralAllOfMetadata struct for ManagedCentralAllOfMetadata
type ManagedCentralAllOfMetadata struct {
	Name                string                                 `json:"name,omitempty"`
	Namespace           string                                 `json:"namespace,omitempty"`
	Internal            bool                                   `json:"internal,omitempty"`
	Annotations         ManagedCentralAllOfMetadataAnnotations `json:"annotations,omitempty"`
	DeletionTimestamp   string                                 `json:"deletionTimestamp,omitempty"`
	SecretsStored       []string                               `json:"secretsStored,omitempty"`
	Secrets             map[string]string                      `json:"secrets,omitempty"`
	SecretDataSha256Sum string                                 `json:"secretDataSha256Sum,omitempty"`
	ExpiredAt           *time.Time                             `json:"expired-at,omitempty"`
}
