package api

import "time"

type ServiceAccount struct {
	ID           string    `json:"id,omitempty"`
	ClientID     string    `json:"clientID,omitempty"`
	ClientSecret string    `json:"clientSecret,omitempty"`
	Name         string    `json:"name,omitempty"`
	CreatedBy    string    `json:"owner,omitempty"`
	Description  string    `json:"description,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
}
