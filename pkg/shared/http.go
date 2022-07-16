package shared

import "net/http"

// CloseResponseBody will close the body of the response, ignoring any errors that may occur.
// It can be used as a defer function.
func CloseResponseBody(resp *http.Response) {
	if resp != nil {
		_ = resp.Body.Close()
	}
}
