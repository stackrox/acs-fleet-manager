package handlers

import (
	"net/http"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"

	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

type dataPlaneClusterHandler struct {
	dataPlaneService services.DataPlaneClusterService
	service          services.ClusterService
}

// NewDataPlaneClusterHandler creates a new instance of dataPlaneClusterHandler
func NewDataPlaneClusterHandler(dataPlaneClusterService services.DataPlaneClusterService, clusterService services.ClusterService) *dataPlaneClusterHandler {
	return &dataPlaneClusterHandler{
		dataPlaneService: dataPlaneClusterService,
		service:          clusterService,
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
			ctx := r.Context()
			dataPlaneClusterStatus, err := presenters.ConvertDataPlaneClusterStatus(dataPlaneClusterUpdateRequest)
			if err != nil {
				return nil, errors.Validation(err.Error())
			}
			svcErr := h.dataPlaneService.UpdateDataPlaneClusterStatus(ctx, dataPlaneClusterID, dataPlaneClusterStatus)
			return nil, svcErr
		},
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

// GetDataPlaneCluster returns current cluster state
func (h *dataPlaneClusterHandler) GetDataPlaneCluster(w http.ResponseWriter, r *http.Request) {
	dataPlaneClusterID := mux.Vars(r)["id"]

	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateLength(&dataPlaneClusterID, "id", &handlers.MinRequiredFieldLength, nil),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			cluster, err := h.service.FindClusterByID(dataPlaneClusterID)
			if err != nil {
				return nil, err
			}
			return presenters.PresentDataPlaneCluster(cluster), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}
