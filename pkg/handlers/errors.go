package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/compat"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"net/http"
	"sort"
	"strconv"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/shared"
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
