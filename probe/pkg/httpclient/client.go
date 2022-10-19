package httpclient

import (
	"net/http"
	"time"
)

// HTTPClient is a http client used to ping the Central UI.
var HTTPClient Client

// Client is a http client interface.
type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

func init() {
	HTTPClient = &http.Client{
		Timeout: 5 * time.Second,
	}
}
