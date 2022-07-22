package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/compat"

	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

// HandlerConfig defines the common things each REST controller must do.
// The corresponding handle() func runs the basic HandlerConfig.
// This is not meant to be an HTTP framework or anything larger than simple CRUD in handlers.
//
//   MarshalInto is a pointer to the object to hold the unmarshaled JSON.
//   Validate is a list of Validation function that run in order, returning fast on the first error.
//   Action is the specific logic a handler must take (e.g, find an object, save an object)
//   ErrorHandler is the way errors are returned to the client
type HandlerConfig struct {
	MarshalInto  interface{}
	Validate     []Validate
	Action       HTTPAction
	ErrorHandler ErrorHandlerFunc
}

// EventStream ...
type EventStream struct {
	ContentType string
	// GetNextEvent should block until there is an event to return.  GetNextEvent should unblock and return io.EOF when if context is canceled.
	GetNextEvent HTTPAction
	Close        func()
}

// Validate ...
type Validate func() *errors.ServiceError

// ErrorHandlerFunc ...
type ErrorHandlerFunc func(r *http.Request, w http.ResponseWriter, err *errors.ServiceError)

// HTTPAction ...
type HTTPAction func() (interface{}, *errors.ServiceError)

func success(r *http.Request) {
	ctx := context.WithValue(r.Context(), logger.ActionResultKey, logger.ActionSuccess)
	ulog := logger.NewUHCLogger(ctx)
	ulog.Infof("operation ended successfully")
}

func errorHandler(r *http.Request, w http.ResponseWriter, cfg *HandlerConfig, err *errors.ServiceError) {
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = shared.HandleError
	}
	ctx := context.WithValue(r.Context(), logger.ActionResultKey, logger.ActionFailed)
	r = r.WithContext(ctx)
	cfg.ErrorHandler(r, w, err)
}

// Handle ...
func Handle(w http.ResponseWriter, r *http.Request, cfg *HandlerConfig, httpStatus int) {
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = shared.HandleError
	}

	if cfg.MarshalInto != nil {

		err := json.NewDecoder(r.Body).Decode(&cfg.MarshalInto)

		// Use the following instead if you want to debug the request body:
		// bytes, err := ioutil.ReadAll(r.Body)
		// if err != nil {
		//	handleError(r.Context(), w, errors.MalformedRequest("Unable to read request body: %s", err))
		//	return
		//}
		// fmt.Println(string(bytes))
		// err = json.Unmarshal(bytes, &cfg.MarshalInto)

		if err != nil {
			errorHandler(r, w, cfg, errors.MalformedRequest("Invalid request format: %s", err))
			return
		}
	}

	for _, v := range cfg.Validate {
		err := v()
		if err != nil {
			errorHandler(r, w, cfg, err)
			return
		}
	}

	result, serviceErr := cfg.Action()

	switch {
	case serviceErr != nil:
		errorHandler(r, w, cfg, serviceErr)
	default:
		shared.WriteJSONResponse(w, httpStatus, result)
		success(r)
	}

}

// HandleDelete ...
func HandleDelete(w http.ResponseWriter, r *http.Request, cfg *HandlerConfig, httpStatus int) {
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = shared.HandleError
	}
	for _, v := range cfg.Validate {
		err := v()
		if err != nil {
			errorHandler(r, w, cfg, err)
			return
		}
	}

	result, serviceErr := cfg.Action()

	switch {
	case serviceErr != nil:
		errorHandler(r, w, cfg, serviceErr)
	default:
		shared.WriteJSONResponse(w, httpStatus, result)
		success(r)
	}

}

// HandleGet ...
func HandleGet(w http.ResponseWriter, r *http.Request, cfg *HandlerConfig) {
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = shared.HandleError
	}

	for _, v := range cfg.Validate {
		err := v()
		if err != nil {
			errorHandler(r, w, cfg, err)
			return
		}
	}

	result, serviceErr := cfg.Action()
	switch {
	case serviceErr == nil:
		shared.WriteJSONResponse(w, http.StatusOK, result)
		success(r)
	default:
		errorHandler(r, w, cfg, serviceErr)
	}
}

// HandleList ...
func HandleList(w http.ResponseWriter, r *http.Request, cfg *HandlerConfig) {
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = shared.HandleError
	}

	ctx := r.Context()
	for _, v := range cfg.Validate {
		err := v()
		if err != nil {
			errorHandler(r, w, cfg, err)
			return
		}
	}

	results, serviceError := cfg.Action()
	if serviceError != nil {
		errorHandler(r, w, cfg, serviceError)
		return
	}

	if stream, ok := results.(EventStream); ok {
		if stream.Close != nil {
			defer stream.Close()
		}
		flusher, ok := w.(http.Flusher)
		if !ok {
			errorHandler(r, w, cfg, errors.BadRequest("streaming unsupported"))
			return
		}

		shared.WriteStreamJSONResponseWithContentType(w, http.StatusOK, nil, stream.ContentType)
		for {
			result, err := stream.GetNextEvent()
			if err != nil {
				ulog := logger.NewUHCLogger(ctx)
				operationID := logger.GetOperationID(ctx)
				// If this is a 400 error, its the user's issue, log as info rather than error
				if err.HTTPCode >= 400 && err.HTTPCode <= 499 {
					ulog.Infof(err.Error())
				} else {
					ulog.Error(err)
				}
				result := compat.WatchEvent{
					Type:  "error",
					Error: ConvertToPrivateError(err.AsOpenapiError(operationID, r.RequestURI)),
				}
				_ = json.NewEncoder(w).Encode(result)
				return
			}
			if result == nil {
				return // the event stream was done.
			}
			_ = json.NewEncoder(w).Encode(result)
			_, _ = fmt.Fprint(w, "\n")
			flusher.Flush() // sends the result to the client (forces Transfer-Encoding: chunked)
		}
	} else {
		shared.WriteJSONResponse(w, http.StatusOK, results)
	}
	success(r)
}

// ConvertToPrivateError ...
func ConvertToPrivateError(e compat.Error) compat.PrivateError {
	return compat.PrivateError{
		Id:          e.Id,
		Kind:        e.Kind,
		Href:        e.Href,
		Code:        e.Code,
		Reason:      e.Reason,
		OperationId: e.OperationId,
	}
}
