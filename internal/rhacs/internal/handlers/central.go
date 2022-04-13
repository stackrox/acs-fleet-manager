package handlers

import (
	"net/http"

	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/config"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/services"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
	"github.com/stackrox/acs-fleet-manager/pkg/services/authorization"
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
	logger.Logger.Infof("Central create request received")

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
	logger.Logger.Infof("Central get request received")
	// return 202 status accepted
	handlers.Handle(w, r, noopHandlerConfig, http.StatusAccepted)
}

func (h centralHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// TODO
	logger.Logger.Infof("Central delete request received")
	// return 202 status accepted
	handlers.Handle(w, r, noopHandlerConfig, http.StatusAccepted)
}

func (h centralHandler) Update(w http.ResponseWriter, r *http.Request) {
	// TODO
	logger.Logger.Infof("Central update request received")
	// return 202 status accepted
	handlers.Handle(w, r, noopHandlerConfig, http.StatusAccepted)
}

func (h centralHandler) List(w http.ResponseWriter, r *http.Request) {
	// TODO
	logger.Logger.Infof("Central list request received")

	// return 202 status accepted
	handlers.Handle(w, r, noopHandlerConfig, http.StatusAccepted)
}

// TODO: complete following internal/dinosaur/internal/handlers/dinosaur.go