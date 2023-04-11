package dbapi

import (
	"time"
)

// CentralDefaultVersion is the type used for entries in the CentralDefaultVersion table
type CentralDefaultVersion struct {
	ID        uint64
	CreatedAt time.Time
	Version   string `json:"version"`
}
