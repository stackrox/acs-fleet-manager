package handlers

import (
	"net/http"

	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/config"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/services"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/services/authorization"

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
	// return 202 status accepted
	handlers.Handle(w, r, noopHandlerConfig, http.StatusAccepted)
}

func (h centralHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// TODO
	logger.Logger.Warningf("Central delete request received")
	// return 202 status accepted
	handlers.Handle(w, r, noopHandlerConfig, http.StatusAccepted)
}

func (h centralHandler) Update(w http.ResponseWriter, r *http.Request) {
	// TODO
	logger.Logger.Warningf("Central update request received")
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
				return nil, errors.NewWithCause(errors.ErrorMalformedRequest, err, "Unable to list dinosaur requests: %s", err.Error())
			}

			centralRequests, _, err := h.service.List(ctx, listArgs)
			if err != nil {
				return nil, err
			}
			centralRequestList := public.CentralRequestList{
				Kind: presenters.CentralRequestListKind,
				NextPageCursor: "",
				Size: 0, 
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