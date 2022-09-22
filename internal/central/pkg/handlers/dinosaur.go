package handlers

import (
	"context"
	"net/http"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
	"github.com/stackrox/acs-fleet-manager/pkg/services/authorization"

	"github.com/gorilla/mux"

	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	coreServices "github.com/stackrox/acs-fleet-manager/pkg/services"
)

type centralHandler struct {
	service        services.CentralService
	providerConfig *config.ProviderConfig
	authService    authorization.Authorization
}

// NewCentralHandler ...
func NewCentralHandler(service services.CentralService, providerConfig *config.ProviderConfig, authService authorization.Authorization) *centralHandler {
	return &centralHandler{
		service:        service,
		providerConfig: providerConfig,
		authService:    authService,
	}
}

func validateCentralResourcesUnspecified(ctx context.Context, centralRequest *public.CentralRequestPayload) handlers.Validate {
	return func() *errors.ServiceError {
		if len(centralRequest.Central.Resources.Limits) > 0 ||
			len(centralRequest.Central.Resources.Requests) > 0 {
			return errors.Forbidden("not allowed to specify central resources")
		}
		return nil
	}
}

func validateScannerResourcesUnspecified(ctx context.Context, centralRequest *public.CentralRequestPayload) handlers.Validate {
	return func() *errors.ServiceError {
		if len(centralRequest.Scanner.Analyzer.Resources.Limits) > 0 ||
			len(centralRequest.Scanner.Analyzer.Resources.Requests) > 0 {
			return errors.Forbidden("not allowed to specify scanner analyzer resources")
		}
		if len(centralRequest.Scanner.Db.Resources.Limits) > 0 ||
			len(centralRequest.Scanner.Db.Resources.Requests) > 0 {
			return errors.Forbidden("not allowed to specify scanner db resources")
		}
		return nil
	}
}

// Create ...
func (h centralHandler) Create(w http.ResponseWriter, r *http.Request) {
	var centralRequest public.CentralRequestPayload
	ctx := r.Context()
	convCentral := &dbapi.CentralRequest{}

	cfg := &handlers.HandlerConfig{
		MarshalInto: &centralRequest,
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "creating central requests"),
			handlers.ValidateLength(&centralRequest.Name, "name", &handlers.MinRequiredFieldLength, &MaxCentralNameLength),
			ValidCentralClusterName(&centralRequest.Name, "name"),
			ValidateCentralClusterNameIsUnique(r.Context(), &centralRequest.Name, h.service),
			ValidateCentralClaims(ctx, &centralRequest, convCentral),
			ValidateCloudProvider(&h.service, convCentral, h.providerConfig, "creating central requests"),
			handlers.ValidateMultiAZEnabled(&centralRequest.MultiAz, "creating central requests"),
			validateCentralResourcesUnspecified(ctx, &centralRequest),
			validateScannerResourcesUnspecified(ctx, &centralRequest),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			svcErr := h.service.RegisterCentralJob(convCentral)
			if svcErr != nil {
				return nil, svcErr
			}
			return presenters.PresentCentralRequest(convCentral), nil
		},
	}

	// return 202 status accepted
	handlers.Handle(w, r, cfg, http.StatusAccepted)
}

// Get ...
func (h centralHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			centralRequest, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			return presenters.PresentCentralRequest(centralRequest), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

// Delete is the handler for deleting a central request
func (h centralHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "deleting central requests"),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()

			err := h.service.RegisterCentralDeprovisionJob(ctx, id)
			return nil, err
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusAccepted)
}

// List ...
func (h centralHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := coreServices.NewListArguments(r.URL.Query())

			if err := listArgs.Validate(); err != nil {
				return nil, errors.NewWithCause(errors.ErrorMalformedRequest, err, "Unable to list central requests: %s", err.Error())
			}

			centralRequests, paging, err := h.service.List(ctx, listArgs)
			if err != nil {
				return nil, err
			}

			centralRequestList := public.CentralRequestList{
				Kind:  "CentralRequestList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []public.CentralRequest{},
			}

			for _, centralRequest := range centralRequests {
				converted := presenters.PresentCentralRequest(centralRequest)
				centralRequestList.Items = append(centralRequestList.Items, converted)
			}

			return centralRequestList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}
