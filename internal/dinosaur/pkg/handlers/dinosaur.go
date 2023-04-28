package handlers

import (
	"net/http"
	"time"

	"github.com/stackrox/acs-fleet-manager/pkg/shared/utils/arrays"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
	"github.com/stackrox/acs-fleet-manager/pkg/services/authorization"

	"github.com/gorilla/mux"

	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	coreServices "github.com/stackrox/acs-fleet-manager/pkg/services"
)

type dinosaurHandler struct {
	service              services.DinosaurService
	providerConfig       *config.ProviderConfig
	authService          authorization.Authorization
	telemetry            *services.Telemetry
	centralRequestConfig *config.CentralRequestConfig
}

// NewDinosaurHandler ...
func NewDinosaurHandler(service services.DinosaurService, providerConfig *config.ProviderConfig,
	authService authorization.Authorization, telemetry *services.Telemetry,
	centralRequestConfig *config.CentralRequestConfig) *dinosaurHandler {
	return &dinosaurHandler{
		service:              service,
		providerConfig:       providerConfig,
		authService:          authService,
		telemetry:            telemetry,
		centralRequestConfig: centralRequestConfig,
	}
}

func validateCentralResourcesUnspecified(dinosaurRequest *public.CentralRequestPayload) handlers.Validate {
	return func() *errors.ServiceError {
		if len(dinosaurRequest.Central.Resources.Limits) > 0 ||
			len(dinosaurRequest.Central.Resources.Requests) > 0 {
			return errors.Forbidden("not allowed to specify central resources")
		}
		return nil
	}
}

func validateScannerResourcesUnspecified(dinosaurRequest *public.CentralRequestPayload) handlers.Validate {
	return func() *errors.ServiceError {
		if len(dinosaurRequest.Scanner.Analyzer.Resources.Limits) > 0 ||
			len(dinosaurRequest.Scanner.Analyzer.Resources.Requests) > 0 {
			return errors.Forbidden("not allowed to specify scanner analyzer resources")
		}
		if len(dinosaurRequest.Scanner.Db.Resources.Limits) > 0 ||
			len(dinosaurRequest.Scanner.Db.Resources.Requests) > 0 {
			return errors.Forbidden("not allowed to specify scanner db resources")
		}
		return nil
	}
}

// Create ...
func (h dinosaurHandler) Create(w http.ResponseWriter, r *http.Request) {
	var centralRequest public.CentralRequestPayload
	ctx := r.Context()
	convCentral := &dbapi.CentralRequest{}

	cfg := &handlers.HandlerConfig{
		MarshalInto: &centralRequest,
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "creating central requests"),
			handlers.ValidateLength(&centralRequest.Name, "name", &handlers.MinRequiredFieldLength, &MaxCentralNameLength),
			ValidDinosaurClusterName(&centralRequest.Name, "name"),
			ValidateDinosaurClusterNameIsUnique(r.Context(), &centralRequest.Name, h.service),
			ValidateDinosaurClaims(ctx, &centralRequest, convCentral),
			ValidateCloudProvider(&h.service, convCentral, h.providerConfig, "creating central requests"),
			handlers.ValidateMultiAZEnabled(&centralRequest.MultiAz, "creating central requests"),
			validateCentralResourcesUnspecified(&centralRequest),
			validateScannerResourcesUnspecified(&centralRequest),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			// Set the central request as internal, **iff** the user agent used within the creation request is contained
			// within the list of user agents for internal services / clients, such as the probe service.
			if arrays.Contains(h.centralRequestConfig.InternalUserAgents, r.UserAgent()) {
				convCentral.Internal = true
			}
			svcErr := h.service.RegisterDinosaurJob(convCentral)
			// Do not track centrals created from internal services.
			if !convCentral.Internal {
				h.telemetry.RegisterTenant(r.Context(), convCentral)
				go func() {
					// This is to raise the chances for the group properties to
					// be procesed by Segment:
					time.Sleep(6 * time.Second)
					h.telemetry.TrackCreationRequested(r.Context(), convCentral.ID, false, svcErr.AsError())
				}()
			}
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
func (h dinosaurHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			dinosaurRequest, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			return presenters.PresentCentralRequest(dinosaurRequest), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

// Delete is the handler for deleting a dinosaur request
func (h dinosaurHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "deleting central requests"),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			centralRequest, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			err = h.service.RegisterDinosaurDeprovisionJob(ctx, id)
			if !centralRequest.Internal {
				h.telemetry.TrackDeletionRequested(ctx, id, false, err.AsError())
			}
			return nil, err
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusAccepted)
}

// List ...
func (h dinosaurHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := coreServices.NewListArguments(r.URL.Query())

			if err := listArgs.Validate(); err != nil {
				return nil, errors.NewWithCause(errors.ErrorMalformedRequest, err, "Unable to list central requests: %s", err.Error())
			}

			dinosaurRequests, paging, err := h.service.List(ctx, listArgs)
			if err != nil {
				return nil, err
			}

			dinosaurRequestList := public.CentralRequestList{
				Kind:  "CentralRequestList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []public.CentralRequest{},
			}

			for _, dinosaurRequest := range dinosaurRequests {
				converted := presenters.PresentCentralRequest(dinosaurRequest)
				dinosaurRequestList.Items = append(dinosaurRequestList.Items, converted)
			}

			return dinosaurRequestList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}
