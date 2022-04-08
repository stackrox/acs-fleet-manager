package handlers

import (
	"net/http"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/services"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"

	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/gorilla/mux"
)

type dataPlaneClusterHandler struct {
	service services.DataPlaneClusterService
}

func NewDataPlaneClusterHandler(service services.DataPlaneClusterService) *dataPlaneClusterHandler {
	return &dataPlaneClusterHandler{
		service: service,
	}
}

func (h *dataPlaneClusterHandler) UpdateDataPlaneClusterStatus(w http.ResponseWriter, r *http.Request) {
	dataPlaneClusterID := mux.Vars(r)["id"]

	var dataPlaneClusterUpdateRequest private.DataPlaneClusterUpdateStatusRequest

	cfg := &handlers.HandlerConfig{
		MarshalInto: &dataPlaneClusterUpdateRequest,
		Validate: []handlers.Validate{
			handlers.ValidateLength(&dataPlaneClusterID, "id", &handlers.MinRequiredFieldLength, nil),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			dataPlaneClusterStatus, err := presenters.ConvertDataPlaneClusterStatus(dataPlaneClusterUpdateRequest)
			if err != nil {
				return nil, errors.Validation(err.Error())
			}
			svcErr := h.service.UpdateDataPlaneClusterStatus(ctx, dataPlaneClusterID, dataPlaneClusterStatus)
			return nil, svcErr
		},
	}

	// TODO do we always to return HTTP 204 No Content?
	handlers.Handle(w, r, cfg, http.StatusNoContent)
}

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
