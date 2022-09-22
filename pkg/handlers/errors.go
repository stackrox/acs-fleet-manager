package handlers

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/stackrox/acs-fleet-manager/internal/central/compat"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"

	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

// NewErrorsHandler ...
func NewErrorsHandler() *ErrorHandler {
	return &ErrorHandler{}
}

// ErrorHandler ...
type ErrorHandler struct{}

var _ RestHandler = ErrorHandler{}

// PresentError ...
func PresentError(err *errors.ServiceError, url string) compat.Error {
	return err.AsOpenapiError("", url)
}

// List ...
func (h ErrorHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			listArgs := services.NewListArguments(r.URL.Query())
			allErrors := errors.Errors()

			// Sort errors by code
			sort.SliceStable(allErrors, func(i, j int) bool {
				return allErrors[i].Code < allErrors[j].Code
			})

			list, total := DetermineListRange(allErrors, listArgs.Page, listArgs.Size)
			errorList := compat.ErrorList{
				Kind:  "ErrorList",
				Page:  int32(listArgs.Page),
				Size:  int32(len(list)),
				Total: int32(total),
				Items: []compat.Error{},
			}
			for _, e := range list {
				err := e.(errors.ServiceError)
				errorList.Items = append(errorList.Items, PresentError(&err, r.RequestURI))
			}

			return errorList, nil
		},
	}

	HandleList(w, r, cfg)
}

// Get ...
func (h ErrorHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			value, err := strconv.Atoi(id)
			if err != nil {
				return nil, errors.NotFound("No error with id %s exists", id)
			}
			code := errors.ServiceErrorCode(value)
			exists, sErr := errors.Find(code)
			if !exists {
				return nil, errors.NotFound("No error with id %s exists", id)
			}
			return PresentError(sErr, r.RequestURI), nil
		},
	}

	HandleGet(w, r, cfg)
}

// Create ...
func (h ErrorHandler) Create(w http.ResponseWriter, r *http.Request) {
	shared.HandleError(r, w, errors.NotImplemented("create"))
}

// Patch ...
func (h ErrorHandler) Patch(w http.ResponseWriter, r *http.Request) {
	shared.HandleError(r, w, errors.NotImplemented("path"))
}

// Delete ...
func (h ErrorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	shared.HandleError(r, w, errors.NotImplemented("delete"))
}
