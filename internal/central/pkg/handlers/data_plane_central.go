package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/gitops"

	"github.com/gorilla/mux"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
)

type dataPlaneCentralHandler struct {
	service              services.DataPlaneCentralService
	centralService       services.CentralService
	clusterService       services.ClusterService
	presenter            *presenters.ManagedCentralPresenter
	gitopsConfigProvider gitops.ConfigProvider
}

// NewDataPlaneCentralHandler ...
func NewDataPlaneCentralHandler(
	service services.DataPlaneCentralService,
	centralService services.CentralService,
	clusterService services.ClusterService,
	presenter *presenters.ManagedCentralPresenter,
	gitopsConfigProvider gitops.ConfigProvider,
) *dataPlaneCentralHandler {
	return &dataPlaneCentralHandler{
		service:              service,
		centralService:       centralService,
		presenter:            presenter,
		gitopsConfigProvider: gitopsConfigProvider,
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
			err := h.service.UpdateDataPlaneCentralService(ctx, clusterID, dataPlaneCentralStatus)
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
			centralRequests, err := h.service.ListByClusterID(clusterID)
			if err != nil {
				return nil, err
			}

			if len(centralRequests) == 0 {
				cluster, err := h.clusterService.FindClusterByID(clusterID)
				if err != nil {
					return nil, err
				}

				if cluster == nil {
					return nil, errors.NotFound("cluster does not exist")
				}
			}

			managedCentralList := private.ManagedCentralList{
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

			managedCentralList.Applications = applicationMaps

			managedCentrals, presentErr := h.presenter.PresentManagedCentrals(r.Context(), centralRequests)
			if presentErr != nil {
				return nil, errors.GeneralError("failed to convert central request to managed central: %v", presentErr)
			}
			managedCentralList.Items = managedCentrals

			return managedCentralList, nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

// GetByID...
func (h *dataPlaneCentralHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	centralID := mux.Vars(r)["id"]
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			centralRequest, svcErr := h.centralService.GetByID(centralID)
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
