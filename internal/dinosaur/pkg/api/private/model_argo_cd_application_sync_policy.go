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

// ArgoCdApplicationSyncPolicy struct for ArgoCdApplicationSyncPolicy
type ArgoCdApplicationSyncPolicy struct {
	Automated                ArgoCdApplicationAutomatedSyncPolicy      `json:"automated,omitempty"`
	SyncOptions              []string                                  `json:"syncOptions,omitempty"`
	ManagedNamespaceMetadata ArgoCdApplicationManagedNamespaceMetadata `json:"managedNamespaceMetadata,omitempty"`
	Retry                    ArgoCdApplicationSyncPolicyRetry          `json:"retry,omitempty"`
}
