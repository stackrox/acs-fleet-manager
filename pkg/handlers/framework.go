package handlers

import (
	"context"
	"encoding/json"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/compat"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/dinosaur_compat"
	"net/http"

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
	Action       HttpAction
	ErrorHandler ErrorHandlerFunc
}

type Validate func() *errors.ServiceError
type ErrorHandlerFunc func(r *http.Request, w http.ResponseWriter, err *errors.ServiceError)
type HttpAction func() (interface{}, *errors.ServiceError)

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

func Handle(w http.ResponseWriter, r *http.Request, cfg *HandlerConfig, httpStatus int) {
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = shared.HandleError
	}

	if cfg.MarshalInto != nil {

		err := json.NewDecoder(r.Body).Decode(&cfg.MarshalInto)

		// Use the following instead if you want to debug the request body:
		//bytes, err := ioutil.ReadAll(r.Body)
		//if err != nil {
		//	handleError(r.Context(), w, errors.MalformedRequest("Unable to read request body: %s", err))
		//	return
		//}
		//fmt.Println(string(bytes))
		//err = json.Unmarshal(bytes, &cfg.MarshalInto)

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

func HandleList(w http.ResponseWriter, r *http.Request, cfg *HandlerConfig) {
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

	results, serviceError := cfg.Action()
	if serviceError != nil {
		errorHandler(r, w, cfg, serviceError)
		return
	}
	shared.WriteJSONResponse(w, http.StatusOK, results)
	success(r)
}
func ConvertToPrivateError(e compat.Error) dinosaur_compat.PrivateError {
	return dinosaur_compat.PrivateError{
		Id:          e.Id,
		Kind:        e.Kind,
		Href:        e.Href,
		Code:        e.Code,
		Reason:      e.Reason,
		OperationId: e.OperationId,
	}
}
