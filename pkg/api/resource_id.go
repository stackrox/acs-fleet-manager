package api

import "github.com/rs/xid"

// NewID ...
func NewID() string {
	return xid.New().String()
}
