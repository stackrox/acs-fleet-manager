package handlers

import (
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/compat"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"net/http"
	"sort"
	"strconv"

	"github.com/stackrox/acs-fleet-manager/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"github.com/gorilla/mux"
)

func NewErrorsHandler() *ErrorHandler {
	return &ErrorHandler{}
}

type ErrorHandler struct{}

var _ RestHandler = ErrorHandler{}

func PresentError(err *errors.ServiceError, url string) compat.Error {
	return err.AsOpenapiError("", url)
}

func (h ErrorHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			listArgs := services.NewListArguments(r.URL.Query())
			allErrors := errors.Errors()

			// Sort errors by code
			sort.SliceStable(allErrors, func(i, j int) bool {
				return allErrors[i].Code < allErrors[j].Code
			})

			errorList := compat.ErrorList{
				Kind:  "ErrorList",
				NextPageCursor: listArgs.NextPageCursor,
				Size:  int32(len(allErrors)),
				Items: []compat.Error{},
			}
			for _, e := range allErrors {
				errorList.Items = append(errorList.Items, PresentError(&e, r.RequestURI))
			}

			return errorList, nil
		},
	}

	HandleList(w, r, cfg)
}

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

func (h ErrorHandler) Create(w http.ResponseWriter, r *http.Request) {
	shared.HandleError(r, w, errors.NotImplemented("create"))
}

func (h ErrorHandler) Patch(w http.ResponseWriter, r *http.Request) {
	shared.HandleError(r, w, errors.NotImplemented("path"))
}

func (h ErrorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	shared.HandleError(r, w, errors.NotImplemented("delete"))
}
