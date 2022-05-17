package handlers

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
	"net/http"

	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/gorilla/mux"
)

type dataPlaneDinosaurHandler struct {
	service         services.DataPlaneDinosaurService
	dinosaurService services.DinosaurService
}

func NewDataPlaneDinosaurHandler(service services.DataPlaneDinosaurService, dinosaurService services.DinosaurService) *dataPlaneDinosaurHandler {
	return &dataPlaneDinosaurHandler{
		service:         service,
		dinosaurService: dinosaurService,
	}
}

func (h *dataPlaneDinosaurHandler) UpdateDinosaurStatuses(w http.ResponseWriter, r *http.Request) {
	clusterId := mux.Vars(r)["id"]
	var data = map[string]private.DataPlaneDinosaurStatus{}

	cfg := &handlers.HandlerConfig{
		MarshalInto: &data,
		Validate:    []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			dataPlaneDinosaurStatus := presenters.ConvertDataPlaneDinosaurStatus(data)
			err := h.service.UpdateDataPlaneDinosaurService(ctx, clusterId, dataPlaneDinosaurStatus)
			return nil, err
		},
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h *dataPlaneDinosaurHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	clusterID := mux.Vars(r)["id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateLength(&clusterID, "id", &handlers.MinRequiredFieldLength, nil),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			managedDinosaurs, err := h.dinosaurService.GetManagedDinosaurByClusterID(clusterID)
			if err != nil {
				return nil, err
			}

			managedDinosaurList := private.ManagedDinosaurList{
				Kind:  "ManagedDinosaurList",
				Items: []private.ManagedDinosaur{},
			}

			for _, mk := range managedDinosaurs {
				converted := presenters.PresentManagedDinosaur(&mk)
				managedDinosaurList.Items = append(managedDinosaurList.Items, converted)
			}
			return managedDinosaurList, nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}
