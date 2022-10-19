package httpclient

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

// MockClient is a fake http client for unit testing.
type MockClient struct {
	StatusCode int
}

// Do mock a http call and always returns a response with the given status code.
func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	json := ``
	r := ioutil.NopCloser(bytes.NewReader([]byte(json)))
	return &http.Response{
		StatusCode: m.StatusCode,
		Body:       r,
	}, nil
}
