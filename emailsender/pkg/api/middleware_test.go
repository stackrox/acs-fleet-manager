package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

var called bool
var nextHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	called = true
})
var testHandler = EnsureJSONContentType(nextHandler)
var req = httptest.NewRequest("GET", "http://testing", nil)

func TestEnsureJSONContentTypeEmptyHeader(t *testing.T) {
	called = false
	testHandler.ServeHTTP(httptest.NewRecorder(), req)

	assert.False(t, called)
}

func TestEnsureJSONContentTypeValid(t *testing.T) {
	called = false
	req.Header.Set("Content-Type", "application/json")
	testHandler.ServeHTTP(httptest.NewRecorder(), req)

	assert.True(t, called)
}

func TestEnsureJSONContentTypeInvalid(t *testing.T) {
	called = false
	req.Header.Set("Content-Type", "broken")
	testHandler.ServeHTTP(httptest.NewRecorder(), req)

	assert.False(t, called)
}
