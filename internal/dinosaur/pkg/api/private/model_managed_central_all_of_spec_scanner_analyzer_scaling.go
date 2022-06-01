/*
 * Red Hat Advanced Cluster Security Service Fleet Manager
 *
 * Red Hat Advanced Cluster Security (RHACS) Service Fleet Manager APIs that are used by internal services e.g fleetshard operators.
 *
 * API version: 1.4.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// ManagedCentralAllOfSpecScannerAnalyzerScaling struct for ManagedCentralAllOfSpecScannerAnalyzerScaling
type ManagedCentralAllOfSpecScannerAnalyzerScaling struct {
	AutoScaling string `json:"autoScaling,omitempty"`
	Replicas    int32  `json:"replicas,omitempty"`
	MinReplicas int32  `json:"minReplicas,omitempty"`
	MaxReplicas int32  `json:"maxReplicas,omitempty"`
}
