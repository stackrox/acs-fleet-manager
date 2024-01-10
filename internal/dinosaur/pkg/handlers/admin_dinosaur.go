// Package handlers ...
package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
	coreServices "github.com/stackrox/acs-fleet-manager/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/services/account"
)

// AdminCentralHandler is the interface for the admin central handler
type AdminCentralHandler interface {
	// Create a central
	Create(w http.ResponseWriter, r *http.Request)
	// Get a central
	Get(w http.ResponseWriter, r *http.Request)
	// List all centrals
	List(w http.ResponseWriter, r *http.Request)
	// Delete a central
	Delete(w http.ResponseWriter, r *http.Request)
	// DbDelete deletes a central from the database
	DbDelete(w http.ResponseWriter, r *http.Request)
	// Restore restores a tenant that was already marked as deleted
	Restore(w http.ResponseWriter, r *http.Request)
	// RotateSecrets rotates secrets within central
	RotateSecrets(w http.ResponseWriter, r *http.Request)
	// PatchExpiredAt sets the expired_at central property
	PatchExpiredAt(w http.ResponseWriter, r *http.Request)
}

type adminCentralHandler struct {
	service        services.DinosaurService
	accountService account.AccountService
	providerConfig *config.ProviderConfig
	telemetry      *services.Telemetry
}

var _ AdminCentralHandler = (*adminCentralHandler)(nil)

// NewAdminCentralHandler ...
func NewAdminCentralHandler(
	service services.DinosaurService,
	accountService account.AccountService,
	providerConfig *config.ProviderConfig,
	telemetry *services.Telemetry,
) AdminCentralHandler {
	return &adminCentralHandler{
		service:        service,
		accountService: accountService,
		providerConfig: providerConfig,
		telemetry:      telemetry,
	}
}

// Create ...
func (h adminCentralHandler) Create(w http.ResponseWriter, r *http.Request) {
	centralRequest := public.CentralRequestPayload{}
	ctx := r.Context()
	convCentral := dbapi.CentralRequest{}

	cfg := &handlers.HandlerConfig{
		MarshalInto: &centralRequest,
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "creating central requests"),
			handlers.ValidateLength(&centralRequest.Name, "name", &handlers.MinRequiredFieldLength, &MaxCentralNameLength),
			ValidDinosaurClusterName(&centralRequest.Name, "name"),
			ValidateDinosaurClusterNameIsUnique(r.Context(), &centralRequest.Name, h.service),
			ValidateDinosaurClaims(ctx, &centralRequest, &convCentral),
			ValidateCloudProvider(&h.service, &convCentral, h.providerConfig, "creating central requests"),
			handlers.ValidateMultiAZEnabled(&centralRequest.MultiAz, "creating central requests"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			svcErr := h.service.RegisterDinosaurJob(ctx, &convCentral)
			h.telemetry.RegisterTenant(ctx, &convCentral, true, svcErr.AsError())

			if svcErr != nil {
				return nil, svcErr
			}
			// TODO(mclasmeier): Do we need PresentDinosaurRequestAdminEndpoint?
			return presenters.PresentCentralRequest(&convCentral), nil
		},
	}

	// return 202 status accepted
	handlers.Handle(w, r, cfg, http.StatusAccepted)
}

// Get ...
func (h adminCentralHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			centralRequest, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			return presenters.PresentDinosaurRequestAdminEndpoint(centralRequest, h.accountService)
		},
	}
	handlers.HandleGet(w, r, cfg)
}

// List ...
func (h adminCentralHandler) List(w http.ResponseWriter, r *http.Request) {
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

			centralRequestList := private.CentralList{
				Kind:  "CentralList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []private.Central{},
			}

			for _, centralRequest := range centralRequests {
				converted, err := presenters.PresentDinosaurRequestAdminEndpoint(centralRequest, h.accountService)
				if err != nil {
					return nil, err
				}

				if converted != nil {
					centralRequestList.Items = append(centralRequestList.Items, *converted)
				}
			}

			return centralRequestList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

// Delete ...
func (h adminCentralHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.service.RegisterDinosaurDeprovisionJob(ctx, id)
			h.telemetry.TrackDeletionRequested(ctx, id, true, err.AsError())
			return nil, err
		},
	}

	handlers.HandleDelete(w, r, cfg, http.StatusAccepted)
}

func (h adminCentralHandler) Restore(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.service.Restore(ctx, id)
			return nil, err
		},
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

// DbDelete implements the endpoint for force-deleting Central tenants in the database in emergency situations requiring manual recovery
// from an inconsistent state.
func (h adminCentralHandler) DbDelete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			centralRequest, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			err = h.service.Delete(centralRequest, true)
			return nil, err
		},
	}

	handlers.HandleDelete(w, r, cfg, http.StatusOK)
}

func (h adminCentralHandler) RotateSecrets(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			updateBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, errors.NewWithCause(errors.ErrorBadRequest, err, "Reading request body: %s", err.Error())
			}

			rotateSecretsRequest := private.CentralRotateSecretsRequest{} // pragma: allowlist secret
			if err := json.Unmarshal(updateBytes, &rotateSecretsRequest); err != nil {
				return nil, errors.NewWithCause(errors.ErrorBadRequest, err, "Unmarshalling request body: %s", err.Error())
			}

			ctx := r.Context()
			centralRequest, svcErr := h.service.Get(ctx, id)
			if svcErr != nil {
				return nil, svcErr
			}
			if rotateSecretsRequest.RotateRhssoClientCredentials {
				svcErr = h.service.RotateCentralRHSSOClient(ctx, centralRequest)
				if svcErr != nil {
					return nil, svcErr
				}
			}
			return nil, nil
		},
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h adminCentralHandler) PatchExpiredAt(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			reason := r.PostFormValue("reason")
			if reason == "" {
				return nil, errors.New(errors.ErrorBadRequest, "No reason provided")
			}

			id := mux.Vars(r)["id"]
			ts := r.PostFormValue("timestamp")
			expired_at := time.Now()
			if ts != "" {
				var err error
				expired_at, err = time.Parse(time.RFC3339, ts)
				if err != nil {
					return nil, errors.NewWithCause(errors.ErrorBadRequest, err, "Cannot parse timestamp: %s", err.Error())
				}
			}
			glog.Warningf("Setting expired_at to %q for central %q: %s", expired_at, id, reason)
			central := &dbapi.CentralRequest{Meta: api.Meta{ID: id}}
			return nil, h.service.Updates(central, map[string]interface{}{
				"expired_at": &expired_at,
			})
		},
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}
