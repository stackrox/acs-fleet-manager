package handlers

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/gitops"
	"github.com/stackrox/acs-fleet-manager/pkg/features"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
	"golang.org/x/sync/errgroup"
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

			if features.TargetedOperatorUpgrades.Enabled() {
				gitopsConfig, err := h.gitopsConfigProvider.Get()
				if err != nil {
					return nil, errors.GeneralError("failed to get GitOps configuration: %v", err)
				}
				managedDinosaurList.RhacsOperators = gitopsConfig.RHACSOperators.ToAPIResponse()
			}

			managedDinosaurList.Items = make([]private.ManagedCentral, len(centralRequests))
			g, ctx := errgroup.WithContext(r.Context())
			const maxParallel = 50
			locks := make(chan struct{}, maxParallel)
			for i := range centralRequests {
				index := i
				g.Go(func() error {
					select {
					case locks <- struct{}{}:
					case <-ctx.Done():
						//nolint:wrapcheck
						return ctx.Err()
					}
					var err error
					managedDinosaurList.Items[index], err = h.presenter.PresentManagedCentral(centralRequests[index])
					<-locks
					//nolint:wrapcheck
					return err
				})
			}
			if err := g.Wait(); err != nil {
				return nil, errors.GeneralError("failed to convert central request to managed central: %v", err)
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
