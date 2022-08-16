package handlers

import (
	"fmt"
	"net/http"

	"github.com/stackrox/acs-fleet-manager/pkg/services/account"

	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
	coreServices "github.com/stackrox/acs-fleet-manager/pkg/services"
)

type adminDinosaurHandler struct {
	service        services.DinosaurService
	accountService account.AccountService
	providerConfig *config.ProviderConfig
}

// NewAdminDinosaurHandler ...
func NewAdminDinosaurHandler(service services.DinosaurService, accountService account.AccountService, providerConfig *config.ProviderConfig) *adminDinosaurHandler {
	return &adminDinosaurHandler{
		service:        service,
		accountService: accountService,
		providerConfig: providerConfig,
	}
}

// Create ...
func (h adminDinosaurHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dinosaurRequest public.CentralRequestPayload
	ctx := r.Context()
	convDinosaur := &dbapi.CentralRequest{}

	cfg := &handlers.HandlerConfig{
		MarshalInto: &dinosaurRequest,
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "creating central requests"),
			handlers.ValidateLength(&dinosaurRequest.Name, "name", &handlers.MinRequiredFieldLength, &MaxDinosaurNameLength),
			ValidDinosaurClusterName(&dinosaurRequest.Name, "name"),
			ValidateDinosaurClusterNameIsUnique(r.Context(), &dinosaurRequest.Name, h.service),
			ValidateDinosaurClaims(ctx, &dinosaurRequest, convDinosaur),
			ValidateCloudProvider(&h.service, convDinosaur, h.providerConfig, "creating central requests"),
			handlers.ValidateMultiAZEnabled(&dinosaurRequest.MultiAz, "creating central requests"),
			ValidateCentralSpec(ctx, &dinosaurRequest, convDinosaur),
			ValidateScannerSpec(ctx, &dinosaurRequest, convDinosaur),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			svcErr := h.service.RegisterDinosaurJob(convDinosaur)
			if svcErr != nil {
				return nil, svcErr
			}
			// TODO(mclasmeier): Do we need PresentDinosaurRequestAdminEndpoint?
			return presenters.PresentDinosaurRequest(convDinosaur), nil
		},
	}

	// return 202 status accepted
	handlers.Handle(w, r, cfg, http.StatusAccepted)
}

// Get ...
func (h adminDinosaurHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			dinosaurRequest, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			return presenters.PresentDinosaurRequestAdminEndpoint(dinosaurRequest, h.accountService)
		},
	}
	handlers.HandleGet(w, r, cfg)
}

// List ...
func (h adminDinosaurHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := coreServices.NewListArguments(r.URL.Query())

			if err := listArgs.Validate(); err != nil {
				return nil, errors.NewWithCause(errors.ErrorMalformedRequest, err, "Unable to list dinosaur requests: %s", err.Error())
			}

			dinosaurRequests, paging, err := h.service.List(ctx, listArgs)
			if err != nil {
				return nil, err
			}

			dinosaurRequestList := private.DinosaurList{
				Kind:  "DinosaurList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []private.Dinosaur{},
			}

			for _, dinosaurRequest := range dinosaurRequests {
				converted, err := presenters.PresentDinosaurRequestAdminEndpoint(dinosaurRequest, h.accountService)
				if err != nil {
					return nil, err
				}

				if converted != nil {
					dinosaurRequestList.Items = append(dinosaurRequestList.Items, *converted)
				}
			}

			return dinosaurRequestList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

// Delete ...
func (h adminDinosaurHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "deleting dinosaur requests"),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()

			err := h.service.RegisterDinosaurDeprovisionJob(ctx, id)
			return nil, err
		},
	}

	handlers.HandleDelete(w, r, cfg, http.StatusAccepted)
}

func updateCentralRequest(request *dbapi.CentralRequest, updateRequest *private.DinosaurUpdateRequest) error {
	if updateRequest == nil {
		return nil
	}

	centralSpec, err := request.GetCentralSpec()
	if err != nil {
		return fmt.Errorf("retrieving CentralSpec from CentralRequest: %w", err)
	}
	scannerSpec, err := request.GetScannerSpec()
	if err != nil {
		return fmt.Errorf("retrieving ScannerSpec from CentralRequest: %w", err)
	}

	err = centralSpec.UpdateFromPrivateAPI(&updateRequest.Central)
	if err != nil {
		return fmt.Errorf("updating CentralSpec from CentralUpdateRequest: %w", err)
	}
	err = scannerSpec.UpdateFromPrivateAPI(&updateRequest.Scanner)
	if err != nil {
		return fmt.Errorf("updating ScannerSpec from CentralUpdateRequest: %w", err)
	}

	new := *request

	err = new.SetCentralSpec(centralSpec)
	if err != nil {
		return fmt.Errorf("updating CentralSpec within CentralRequest: %w", err)
	}

	err = new.SetScannerSpec(scannerSpec)
	if err != nil {
		return fmt.Errorf("updating ScannerSpec within CentralRequest: %w", err)
	}

	// Disabled this for now, since it is unclear as of now what our specific requirements
	// and dependencies are for this to work.
	//
	// TODO(create-ticket): Evaluate use-case and potentially enable version updating.
	//
	// if updateRequest.DinosaurOperatorVersion != "" {
	// 	new.DesiredCentralOperatorVersion = updateRequest.DinosaurOperatorVersion
	// }

	// if updateRequest.DinosaurVersion != "" {
	// 	new.DesiredCentralVersion = updateRequest.DinosaurVersion
	// }

	*request = new
	return nil
}

// Update a Central instance.
func (h adminDinosaurHandler) Update(w http.ResponseWriter, r *http.Request) {

	var dinosaurUpdateReq private.DinosaurUpdateRequest
	cfg := &handlers.HandlerConfig{
		MarshalInto: &dinosaurUpdateReq,
		Validate:    []handlers.Validate{},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			dinosaurRequest, svcErr := h.service.Get(ctx, id)
			if svcErr != nil {
				return nil, svcErr
			}

			err := updateCentralRequest(dinosaurRequest, &dinosaurUpdateReq)
			if err != nil {
				return nil, errors.NewWithCause(errors.ErrorBadRequest, err, "Updating CentralRequest")
			}

			svcErr = h.service.VerifyAndUpdateDinosaurAdmin(ctx, dinosaurRequest)
			if svcErr != nil {
				return nil, svcErr
			}
			return presenters.PresentDinosaurRequestAdminEndpoint(dinosaurRequest, h.accountService)
		},
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}
