/*
 * Red Hat Advanced Cluster Security Service Fleet Manager Admin API
 *
 * Red Hat Advanced Cluster Security (RHACS) Service Fleet Manager Admin APIs that can be used by RHACS Managed Service Operations Team.
 *
 * API version: 0.0.3
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// CentralUpdateRequest struct for CentralUpdateRequest
type CentralUpdateRequest struct {
	CentralOperatorVersion string      `json:"central_operator_version,omitempty"`
	CentralVersion         string      `json:"central_version,omitempty"`
	Central                CentralSpec `json:"central,omitempty"`
	Scanner                ScannerSpec `json:"scanner,omitempty"`
}
