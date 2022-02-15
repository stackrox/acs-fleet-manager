package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
)

// SendNotFound sends a 404 response with some details about the non existing resource.
func SendNotFound(w http.ResponseWriter, r *http.Request) {
	// Set the content type:
	w.Header().Set("Content-Type", "application/json")

	// Prepare the body:
	reason := fmt.Sprintf(
		"The requested resource '%s' doesn't exist",
		r.URL.Path,
	)
	apiError := errors.NotFound(reason).AsOpenapiError("", r.RequestURI)
	data, err := json.Marshal(apiError)
	if err != nil {
		SendPanic(w, r)
		return
	}

	// Send the response:
	w.WriteHeader(http.StatusNotFound)
	_, err = w.Write(data)
	if err != nil {
		err = fmt.Errorf("Can't send response body for request '%s'", r.URL.Path)
		glog.Error(err)
		sentry.CaptureException(err)
		return
	}
}

// SendMethodNotAllowed response
func SendMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	reason := fmt.Sprintf("Method: %s is not allowed or not yet implemented for %s", r.Method, r.URL.Path)
	apiError := errors.NotImplemented(reason).AsOpenapiError("", r.RequestURI)
	jsonPayload, err := json.Marshal(apiError)
	if err != nil {
		SendPanic(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusMethodNotAllowed)
	_, err = w.Write(jsonPayload)
	if err != nil {
		err = fmt.Errorf("Can't send response body for request '%s'", r.URL.Path)
		glog.Error(err)
		sentry.CaptureException(err)
		return
	}
}

// SendUnauthorized response
func SendUnauthorized(w http.ResponseWriter, r *http.Request, message string) {
	w.Header().Set("Content-Type", "application/json")

	// Prepare the body:
	apiError := errors.Unauthorized(message).AsOpenapiError("", r.RequestURI)
	data, err := json.Marshal(apiError)
	if err != nil {
		SendPanic(w, r)
		return
	}

	// Send the response:
	w.WriteHeader(http.StatusUnauthorized)
	_, err = w.Write(data)
	if err != nil {
		err = fmt.Errorf("Can't send response body for request '%s'", r.URL.Path)
		glog.Error(err)
		sentry.CaptureException(err)
		return
	}
}

// SendPanic sends a panic error response to the client, but it doesn't end the process.
func SendPanic(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_, err := w.Write(panicBody)
	if err != nil {
		err = fmt.Errorf(
			"Can't send panic response for request '%s': %s",
			r.URL.Path,
			err.Error(),
		)
		glog.Error(err)
		sentry.CaptureException(err)
	}
}

// panicBody is the error body that will be sent when something unexpected happens while trying to
// send another error response. For example, if sending an error response fails because the error
// description can't be converted to JSON.
var panicBody []byte

func init() {
	var err error

	// Create the panic error body:
	apiError := errors.UnableToSendErrorResponse().AsOpenapiError("", "")
	panicBody, err = json.Marshal(apiError)
	if err != nil {
		err = fmt.Errorf(
			"Can't create the panic error body: %s",
			err.Error(),
		)
		glog.Error(err)
		sentry.CaptureException(err)
		os.Exit(1)
	}
}
