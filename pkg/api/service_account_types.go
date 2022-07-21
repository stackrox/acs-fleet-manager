package api

import "time"

// ServiceAccount ...
type ServiceAccount struct {
	ID           string    `json:"id,omitempty"`
	ClientID     string    `json:"clientID,omitempty"`
	ClientSecret string    `json:"clientSecret,omitempty"`
	Name         string    `json:"name,omitempty"`
	Description  string    `json:"description,omitempty"`
	CreatedBy    string    `json:"created_by,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
}
