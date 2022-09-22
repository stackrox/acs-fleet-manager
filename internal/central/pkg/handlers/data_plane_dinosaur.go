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

type dataPlaneCentralHandler struct {
	service        services.DataPlaneDinosaurService
	centralService services.CentralService
	presenter      *presenters.ManagedCentralPresenter
}

// NewDataPlaneCentralHandler ...
func NewDataPlaneCentralHandler(service services.DataPlaneDinosaurService, centralService services.CentralService, presenter *presenters.ManagedCentralPresenter) *dataPlaneCentralHandler {
	return &dataPlaneCentralHandler{
		service:        service,
		centralService: centralService,
		presenter:      presenter,
	}
}

// UpdateCentralStatuses ...
func (h *dataPlaneCentralHandler) UpdateCentralStatuses(w http.ResponseWriter, r *http.Request) {
	clusterID := mux.Vars(r)["id"]
	var data = map[string]private.DataPlaneCentralStatus{}

	cfg := &handlers.HandlerConfig{
		MarshalInto: &data,
		Validate:    []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			dataPlaneCentralStatus := presenters.ConvertDataPlaneCentralStatus(data)
			err := h.service.UpdateDataPlaneDinosaurService(ctx, clusterID, dataPlaneCentralStatus)
			return nil, err
		},
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

// GetAll ...
func (h *dataPlaneCentralHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	clusterID := mux.Vars(r)["id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateLength(&clusterID, "id", &handlers.MinRequiredFieldLength, nil),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			centralRequests, err := h.centralService.ListByClusterID(clusterID)
			if err != nil {
				return nil, err
			}

			managedCentralList := private.ManagedCentralList{
				Kind:  "ManagedCentralList",
				Items: []private.ManagedCentral{},
			}

			for i := range centralRequests {
				converted := h.presenter.PresentManagedCentral(centralRequests[i])
				managedCentralList.Items = append(managedCentralList.Items, converted)
			}
			return managedCentralList, nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}
