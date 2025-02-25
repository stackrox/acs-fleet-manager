package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/gitops"

	"github.com/gorilla/mux"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
)

type dataPlaneDinosaurHandler struct {
	service              services.DataPlaneCentralService
	dinosaurService      services.DinosaurService
	presenter            *presenters.ManagedCentralPresenter
	gitopsConfigProvider gitops.ConfigProvider
}

// NewDataPlaneDinosaurHandler ...
func NewDataPlaneDinosaurHandler(
	service services.DataPlaneCentralService,
	dinosaurService services.DinosaurService,
	presenter *presenters.ManagedCentralPresenter,
	gitopsConfigProvider gitops.ConfigProvider,
) *dataPlaneDinosaurHandler {
	return &dataPlaneDinosaurHandler{
		service:              service,
		dinosaurService:      dinosaurService,
		presenter:            presenter,
		gitopsConfigProvider: gitopsConfigProvider,
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
			centralRequests, err := h.service.ListByClusterID(clusterID)
			if err != nil {
				return nil, err
			}

			managedDinosaurList := private.ManagedCentralList{
				Kind:  "ManagedCentralList",
				Items: []private.ManagedCentral{},
			}

			gitopsConfig, gitopsConfigErr := h.gitopsConfigProvider.Get()
			if gitopsConfigErr != nil {
				return nil, errors.GeneralError("failed to get GitOps configuration: %v", gitopsConfigErr)
			}

			applicationMaps := make([]map[string]interface{}, 0, len(gitopsConfig.Applications))
			for _, app := range gitopsConfig.Applications {
				jsonBytes, err := json.Marshal(app)
				if err != nil {
					return nil, errors.GeneralError("failed to marshal application: %v", err)
				}
				applicationMap := map[string]interface{}{}
				if err := json.Unmarshal(jsonBytes, &applicationMap); err != nil {
					return nil, errors.GeneralError("failed to unmarshal application: %v", err)
				}
				applicationMaps = append(applicationMaps, applicationMap)
			}

			managedDinosaurList.Applications = applicationMaps

			managedCentrals, presentErr := h.presenter.PresentManagedCentrals(r.Context(), centralRequests)
			if presentErr != nil {
				return nil, errors.GeneralError("failed to convert central request to managed central: %v", presentErr)
			}
			managedDinosaurList.Items = managedCentrals

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
			centralRequest, svcErr := h.dinosaurService.GetByID(centralID)
			if svcErr != nil {
				return nil, svcErr
			}

			converted, err := h.presenter.PresentManagedCentralWithSecrets(centralRequest)
			if err != nil {
				return nil, errors.GeneralError("failed to convert central request to managed central: %v", err)
			}

			return converted, nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}
