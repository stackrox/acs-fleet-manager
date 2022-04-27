package handlers

import (
	"net/http"

	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/config"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/services"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/services/authorization"

	"github.com/gorilla/mux"

	coreServices "github.com/stackrox/acs-fleet-manager/pkg/services"
)

var (
	noopHandlerConfig = &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			return nil, nil
		},
	}
)

type centralHandler struct {
	service        services.CentralService
	providerConfig *config.ProviderConfig
	authService    authorization.Authorization
}

func NewCentralHandler(service services.CentralService, providerConfig *config.ProviderConfig, authService authorization.Authorization) *centralHandler {
	return &centralHandler{
		service:        service,
		providerConfig: providerConfig,
		authService:    authService,
	}
}

func (h centralHandler) Create(w http.ResponseWriter, r *http.Request) {
	// TODO
	logger.Logger.Warningf("Central create request received")

	var request public.CentralRequestPayload
	cfg := &handlers.HandlerConfig{
		MarshalInto: request,
		Action: func() (interface{}, *errors.ServiceError) {
			return nil, nil
		},
	}
	// return 202 status accepted
	handlers.Handle(w, r, cfg, http.StatusAccepted)
}


func (h centralHandler) Get(w http.ResponseWriter, r *http.Request) {
	// TODO
	logger.Logger.Warningf("Central get request received")
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			request, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			return presenters.PresentCentralRequest(request), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h centralHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// TODO
	logger.Logger.Warningf("Central delete request received")
	// return 202 status accepted
	handlers.Handle(w, r, noopHandlerConfig, http.StatusAccepted)
}

func (h centralHandler) Update(w http.ResponseWriter, r *http.Request) {
	// TODO
	// TODO take https://github.com/bf2fc6cc711aee1a0c2a/ffm-fleet-manager-go-template/pull/37 and
	// https://bf2.zulipchat.com/#narrow/stream/315461-factorized-fleet-manager/topic/How.20is.20listing.20kafkas.20restricted.20by.20customer.3F/near/280317885
	// into account when implementing authorization

	logger.Logger.Warningf("Central update request received")
	var updateRequest public.CentralUpdateRequest
	id := mux.Vars(r)["id"]
	ctx := r.Context()
	tenantRequest, tenantGetError := h.service.Get(ctx, id)
	validateTenantFound := func() handlers.Validate {
		return func() *errors.ServiceError {
			return tenantGetError
		}
	}
	cfg := &handlers.HandlerConfig{
		MarshalInto: &updateRequest,
		Validate: []handlers.Validate{
			validateTenantFound(),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			// TODO implement update logic
			var updateTenantRequest *dbapi.CentralRequest
			svcErr := h.service.Update(updateTenantRequest)
			if svcErr != nil {
				return nil, svcErr
			}

			// FIXME: tenantRequest should be updated according to the request params
			return presenters.PresentCentralRequest(tenantRequest), nil
		},
	}
	handlers.Handle(w, r, cfg, http.StatusOK)


	// return 202 status accepted
	handlers.Handle(w, r, noopHandlerConfig, http.StatusAccepted)
}

func (h centralHandler) List(w http.ResponseWriter, r *http.Request) {
	// TODO 
	logger.Logger.Warningf("Central list request received")

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
				Kind: presenters.CentralRequestListKind,
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Items: []public.CentralRequest{},
			}
			for _, centralRequest := range centralRequests {
				presentedCentral := presenters.PresentCentralRequest(centralRequest)
				centralRequestList.Items = append(centralRequestList.Items, presentedCentral)
			}

			return centralRequestList, nil
		},
	}
	handlers.HandleList(w, r, cfg)
}

// TODO: complete following internal/dinosaur/internal/handlers/dinosaur.go