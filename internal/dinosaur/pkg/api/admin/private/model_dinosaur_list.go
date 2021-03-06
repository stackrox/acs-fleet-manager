/*
 * Dinosaur Service Fleet Manager Admin APIs
 *
 * The admin APIs for the fleet manager of Dinosaur service
 *
 * API version: 0.0.2
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// DinosaurList struct for DinosaurList
type DinosaurList struct {
	Kind  string     `json:"kind"`
	Page  int32      `json:"page"`
	Size  int32      `json:"size"`
	Total int32      `json:"total"`
	Items []Dinosaur `json:"items"`
}
