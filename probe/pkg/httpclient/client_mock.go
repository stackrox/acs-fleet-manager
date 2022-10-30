package httpclient

import "net/http"

// RoundTripFunc declares a type for the round trip function.
type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip mocks the round trip of the http client.
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// NewMockClient returns *http.Client with a mocked Transport.
func NewMockClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(fn),
	}
}
