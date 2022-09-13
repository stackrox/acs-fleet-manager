package client

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractCentralError(t *testing.T) {
	// 1. Error in returned response
	json := `{"error":"error-message"}`
	r := ioutil.NopCloser(strings.NewReader(json))
	response := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       r,
	}
	reason := extractCentralError(response)
	assert.Equal(t, "error-message", reason)

	// 2. Empty body results in general reason
	response = &http.Response{
		StatusCode: http.StatusBadRequest,
	}
	reason = extractCentralError(response)
	assert.Equal(t, couldNotParseReason, reason)

	// 3. Incorrect JSON results in general reason
	json = `{"error":"error-message`
	r = ioutil.NopCloser(strings.NewReader(json))
	response = &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       r,
	}
	reason = extractCentralError(response)
	assert.Equal(t, couldNotParseReason, reason)

	// 4. No error in response
	json = `{"id":"error-message"}`
	r = ioutil.NopCloser(strings.NewReader(json))
	response = &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       r,
	}
	reason = extractCentralError(response)
	assert.Equal(t, couldNotParseReason, reason)

}
