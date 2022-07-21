package observatorium

import (
	"time"
)

// Configuration ...
type Configuration struct {
	BaseURL   string
	AuthToken string
	Cookie    string
	Timeout   time.Duration
	Debug     bool
	Insecure  bool
	AuthType  string
}
