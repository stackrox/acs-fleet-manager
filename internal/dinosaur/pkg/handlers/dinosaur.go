package handlers

import (
	"context"
	"net/http"

	goerr "github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
	"github.com/stackrox/acs-fleet-manager/pkg/services/authorization"

	"github.com/gorilla/mux"

	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	coreServices "github.com/stackrox/acs-fleet-manager/pkg/services"
)

type dinosaurHandler struct {
	service        services.DinosaurService
	providerConfig *config.ProviderConfig
	authService    authorization.Authorization
	telemetry      *services.Telemetry
}

// NewDinosaurHandler ...
func NewDinosaurHandler(service services.DinosaurService, providerConfig *config.ProviderConfig, authService authorization.Authorization, telemetry *services.Telemetry) *dinosaurHandler {
	return &dinosaurHandler{
		service:        service,
		providerConfig: providerConfig,
		authService:    authService,
		telemetry:      telemetry,
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
	convDinosaur := &dbapi.CentralRequest{}

	cfg := &handlers.HandlerConfig{
		MarshalInto: &centralRequest,
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "creating central requests"),
			handlers.ValidateLength(&centralRequest.Name, "name", &handlers.MinRequiredFieldLength, &MaxDinosaurNameLength),
			ValidDinosaurClusterName(&centralRequest.Name, "name"),
			ValidateDinosaurClusterNameIsUnique(r.Context(), &centralRequest.Name, h.service),
			ValidateDinosaurClaims(ctx, &centralRequest, convDinosaur),
			ValidateCloudProvider(&h.service, convDinosaur, h.providerConfig, "creating central requests"),
			handlers.ValidateMultiAZEnabled(&centralRequest.MultiAz, "creating central requests"),
			validateCentralResourcesUnspecified(&centralRequest),
			validateScannerResourcesUnspecified(&centralRequest),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			// Set the internalInstance to true. This will mean that telemetry won't be enabled for it, both within the
			// central itself and the track requests.
			// We probably want to expose this as a configuration setting to change it in a "flexible" way in the future.
			if r.UserAgent() == "fleet-manager-probe-service" {
				convDinosaur.Internal = true
			}
			svcErr := h.service.RegisterDinosaurJob(convDinosaur)

			// Do not track centrals created from internal services.
			if !convDinosaur.Internal {
				h.telemetry.RegisterTenant(r.Context(), convDinosaur)
				h.telemetry.TrackCreationRequested(r.Context(), convDinosaur.ID, false, svcErr.AsError())
			}
			if svcErr != nil {
				return nil, svcErr
			}
			return presenters.PresentCentralRequest(convDinosaur), nil
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
			err := h.service.RegisterDinosaurDeprovisionJob(ctx, id)
			h.telemetry.TrackDeletionRequested(ctx, id, false, err.AsError())
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

func getAccountIDFromContext(ctx context.Context) (string, error) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return "", goerr.Wrap(err, "cannot obtain claims from context")
	}
	accountID, err := claims.GetAccountID()
	if err != nil {
		return "", goerr.Wrap(err, "no account id in claims")
	}
	return accountID, nil
}
