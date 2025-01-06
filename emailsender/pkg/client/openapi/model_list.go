/*
 * Red Hat Advanced Cluster Security Service Email Sender
 *
 * Red Hat Advanced Cluster Security (RHACS) Email Sender service allows sending email notification from ACS Central tenants without bringing an own SMTP service.
 *
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech). DO NOT EDIT.
package openapi

// List struct for List
type List struct {
	Kind  string `json:"kind"`
	Page  int32  `json:"page"`
	Size  int32  `json:"size"`
	Total int32  `json:"total"`
}