package handlers

import (
	"net/http"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"

	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

type dataPlaneClusterHandler struct {
	service services.DataPlaneClusterService
}

// NewDataPlaneClusterHandler ...
func NewDataPlaneClusterHandler(service services.DataPlaneClusterService) *dataPlaneClusterHandler {
	return &dataPlaneClusterHandler{
		service: service,
	}
}

// UpdateDataPlaneClusterStatus ...
func (h *dataPlaneClusterHandler) UpdateDataPlaneClusterStatus(w http.ResponseWriter, r *http.Request) {
	dataPlaneClusterID := mux.Vars(r)["id"]

	var dataPlaneClusterUpdateRequest private.DataPlaneClusterUpdateStatusRequest

	cfg := &handlers.HandlerConfig{
		MarshalInto: &dataPlaneClusterUpdateRequest,
		Validate: []handlers.Validate{
			handlers.ValidateLength(&dataPlaneClusterID, "id", &handlers.MinRequiredFieldLength, nil),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			dataPlaneClusterStatus := presenters.ConvertDataPlaneClusterStatus(dataPlaneClusterUpdateRequest)
			svcErr := h.service.UpdateDataPlaneClusterStatus(dataPlaneClusterID, dataPlaneClusterStatus)
			return nil, svcErr
		},
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

// GetDataPlaneClusterConfig ...
func (h *dataPlaneClusterHandler) GetDataPlaneClusterConfig(w http.ResponseWriter, r *http.Request) {
	dataPlaneClusterID := mux.Vars(r)["id"]

	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateLength(&dataPlaneClusterID, "id", &handlers.MinRequiredFieldLength, nil),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			dataClusterConfig, err := h.service.GetDataPlaneClusterConfig(ctx, dataPlaneClusterID)
			if err != nil {
				return nil, err
			}
			return presenters.PresentDataPlaneClusterConfig(dataClusterConfig), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}
