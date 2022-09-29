/*
 * Red Hat Advanced Cluster Security Service Fleet Manager
 *
 * Red Hat Advanced Cluster Security (RHACS) Service Fleet Manager is a Rest API to manage instances of ACS components.
 *
 * API version: 1.2.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech). DO NOT EDIT.
package public

// ScannerSpec struct for ScannerSpec
type ScannerSpec struct {
	Analyzer ScannerSpecAnalyzer `json:"analyzer,omitempty"`
	Db       ScannerSpecDb       `json:"db,omitempty"`
}
