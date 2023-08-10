package handlers

import (
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/operator"
	"github.com/stackrox/acs-fleet-manager/pkg/features"
	"net/http"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"

	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

type dataPlaneDinosaurHandler struct {
	service         services.DataPlaneCentralService
	dinosaurService services.DinosaurService
	presenter       *presenters.ManagedCentralPresenter
}

// NewDataPlaneDinosaurHandler ...
func NewDataPlaneDinosaurHandler(service services.DataPlaneCentralService, dinosaurService services.DinosaurService, presenter *presenters.ManagedCentralPresenter) *dataPlaneDinosaurHandler {
	return &dataPlaneDinosaurHandler{
		service:         service,
		dinosaurService: dinosaurService,
		presenter:       presenter,
	}
}

// UpdateDinosaurStatuses ...
func (h *dataPlaneDinosaurHandler) UpdateDinosaurStatuses(w http.ResponseWriter, r *http.Request) {
	clusterID := mux.Vars(r)["id"]
	var data = map[string]private.DataPlaneCentralStatus{}

	cfg := &handlers.HandlerConfig{
		MarshalInto: &data,
		Validate:    []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			dataPlaneDinosaurStatus := presenters.ConvertDataPlaneDinosaurStatus(data)
			err := h.service.UpdateDataPlaneCentralService(ctx, clusterID, dataPlaneDinosaurStatus)
			return nil, err
		},
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

// GetAll ...
func (h *dataPlaneDinosaurHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	clusterID := mux.Vars(r)["id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateLength(&clusterID, "id", &handlers.MinRequiredFieldLength, nil),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			centralRequests, err := h.dinosaurService.ListByClusterID(clusterID)
			if err != nil {
				return nil, err
			}

			managedDinosaurList := private.ManagedCentralList{
				Kind:  "ManagedCentralList",
				Items: []private.ManagedCentral{},
			}

			// TODO: check that the correct GitOps configuration is added to the response
			if features.TargetedOperatorUpgrades.Enabled() {
				managedDinosaurList.RhacsOperators = operator.GetConfig().ToAPIResponse()
			}

			for i := range centralRequests {
				converted := h.presenter.PresentManagedCentral(centralRequests[i])
				managedDinosaurList.Items = append(managedDinosaurList.Items, converted)
			}
			return managedDinosaurList, nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

// GetByID...
func (h *dataPlaneDinosaurHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	centralID := mux.Vars(r)["id"]
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			centralRequest, err := h.dinosaurService.GetByID(centralID)
			if err != nil {
				return nil, err
			}

			converted := h.presenter.PresentManagedCentralWithSecrets(centralRequest)

			return converted, nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}
