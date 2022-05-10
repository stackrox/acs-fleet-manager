/*
 * Red Hat Advanced Cluster Security Service Fleet Manager
 *
 * Red Hat Advanced Cluster Security (RHACS) Service Fleet Manager is a Rest API to manage instances of ACS components.
 *
 * API version: 1.2.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

// MetricsRangeQueryList struct for MetricsRangeQueryList
type MetricsRangeQueryList struct {
	Kind  string       `json:"kind,omitempty"`
	Id    string       `json:"id,omitempty"`
	Items []RangeQuery `json:"items,omitempty"`
}
