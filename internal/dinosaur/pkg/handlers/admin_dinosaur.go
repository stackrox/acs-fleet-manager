// Package handlers ...
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/converters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/defaults"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
	coreServices "github.com/stackrox/acs-fleet-manager/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/services/account"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// AdminCentralHandler is the interface for the admin central handler
type AdminCentralHandler interface {
	// Create a central
	Create(w http.ResponseWriter, r *http.Request)
	// Get a central
	Get(w http.ResponseWriter, r *http.Request)
	// List all centrals
	List(w http.ResponseWriter, r *http.Request)
	// Update a central
	Update(w http.ResponseWriter, r *http.Request)
	// Delete a central
	Delete(w http.ResponseWriter, r *http.Request)
	// DbDelete deletes a central from the database
	DbDelete(w http.ResponseWriter, r *http.Request)
	// SetCentralDefaultVersion sets the default version for a central
	SetCentralDefaultVersion(w http.ResponseWriter, r *http.Request)
	// GetCentralDefaultVersion gets the default version for a central
	GetCentralDefaultVersion(w http.ResponseWriter, r *http.Request)
	// Restore restores a tenant that was already marked as deleted
	Restore(w http.ResponseWriter, r *http.Request)
}

type adminCentralHandler struct {
	service                      services.DinosaurService
	accountService               account.AccountService
	providerConfig               *config.ProviderConfig
	telemetry                    *services.Telemetry
	centralDefaultVersionService services.CentralDefaultVersionService
}

var _ AdminCentralHandler = (*adminCentralHandler)(nil)

// NewAdminCentralHandler ...
func NewAdminCentralHandler(
	service services.DinosaurService,
	accountService account.AccountService,
	providerConfig *config.ProviderConfig,
	telemetry *services.Telemetry,
	centralDefaultVersionService services.CentralDefaultVersionService) AdminCentralHandler {
	return &adminCentralHandler{
		service:                      service,
		accountService:               accountService,
		providerConfig:               providerConfig,
		telemetry:                    telemetry,
		centralDefaultVersionService: centralDefaultVersionService,
	}
}

// Create ...
func (h adminCentralHandler) Create(w http.ResponseWriter, r *http.Request) {
	centralRequest := public.CentralRequestPayload{
		Central: public.CentralSpec{
			Resources: converters.ConvertCoreV1ResourceRequirementsToPublic(&defaults.CentralResources),
		},
		Scanner: public.ScannerSpec{
			Analyzer: public.ScannerSpecAnalyzer{
				Resources: converters.ConvertCoreV1ResourceRequirementsToPublic(&defaults.ScannerAnalyzerResources),
				Scaling:   converters.ConvertScalingToPublic(&dbapi.DefaultScannerAnalyzerScaling),
			},
			Db: public.ScannerSpecDb{
				Resources: converters.ConvertCoreV1ResourceRequirementsToPublic(&defaults.ScannerDbResources),
			},
		},
	}
	ctx := r.Context()
	convCentral := dbapi.CentralRequest{}

	cfg := &handlers.HandlerConfig{
		MarshalInto: &centralRequest,
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "creating central requests"),
			handlers.ValidateLength(&centralRequest.Name, "name", &handlers.MinRequiredFieldLength, &MaxCentralNameLength),
			ValidDinosaurClusterName(&centralRequest.Name, "name"),
			ValidateDinosaurClusterNameIsUnique(r.Context(), &centralRequest.Name, h.service),
			ValidateDinosaurClaims(ctx, &centralRequest, &convCentral),
			ValidateCloudProvider(&h.service, &convCentral, h.providerConfig, "creating central requests"),
			handlers.ValidateMultiAZEnabled(&centralRequest.MultiAz, "creating central requests"),
			ValidateCentralSpec(ctx, &centralRequest, &convCentral),
			ValidateScannerSpec(ctx, &centralRequest, &convCentral),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			svcErr := h.service.RegisterDinosaurJob(&convCentral)
			h.telemetry.RegisterTenant(ctx, &convCentral, true, svcErr.AsError())

			if svcErr != nil {
				return nil, svcErr
			}
			// TODO(mclasmeier): Do we need PresentDinosaurRequestAdminEndpoint?
			return presenters.PresentCentralRequest(&convCentral), nil
		},
	}

	// return 202 status accepted
	handlers.Handle(w, r, cfg, http.StatusAccepted)
}

// Get ...
func (h adminCentralHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			centralRequest, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			return presenters.PresentDinosaurRequestAdminEndpoint(centralRequest, h.accountService)
		},
	}
	handlers.HandleGet(w, r, cfg)
}

// List ...
func (h adminCentralHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := coreServices.NewListArguments(r.URL.Query())

			if err := listArgs.Validate(); err != nil {
				return nil, errors.NewWithCause(errors.ErrorMalformedRequest, err, "Unable to list central requests: %s", err.Error())
			}

			centralRequests, paging, err := h.service.List(ctx, listArgs)
			if err != nil {
				return nil, err
			}

			centralRequestList := private.CentralList{
				Kind:  "CentralList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []private.Central{},
			}

			for _, centralRequest := range centralRequests {
				converted, err := presenters.PresentDinosaurRequestAdminEndpoint(centralRequest, h.accountService)
				if err != nil {
					return nil, err
				}

				if converted != nil {
					centralRequestList.Items = append(centralRequestList.Items, *converted)
				}
			}

			return centralRequestList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

// Delete ...
func (h adminCentralHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.service.RegisterDinosaurDeprovisionJob(ctx, id)
			h.telemetry.TrackDeletionRequested(ctx, id, true, err.AsError())
			return nil, err
		},
	}

	handlers.HandleDelete(w, r, cfg, http.StatusAccepted)
}

func (h adminCentralHandler) Restore(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.service.Restore(ctx, id)
			return nil, err
		},
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

// DbDelete implements the endpoint for force-deleting Central tenants in the database in emergency situations requiring manual recovery
// from an inconsistent state.
func (h adminCentralHandler) DbDelete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			centralRequest, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			err = h.service.Delete(centralRequest, true)
			return nil, err
		},
	}

	handlers.HandleDelete(w, r, cfg, http.StatusOK)
}

func validateResourcesList(rl *corev1.ResourceList) error {
	if rl == nil {
		return nil
	}
	for name := range *rl {
		_, isSupported := validateResourceName(name)
		if !isSupported {
			return fmt.Errorf("resource type %q is not supported", name)
		}
	}
	return nil
}

func validateCoreV1Resources(to *corev1.ResourceRequirements) error {
	newResources := to.DeepCopy()

	err := validateResourcesList(&newResources.Limits)
	if err != nil {
		return err
	}
	err = validateResourcesList(&newResources.Requests)
	if err != nil {
		return err
	}

	*to = *newResources
	return nil
}

// validateCentralSpec validates the CentralSpec using the non-zero fields from the API's CentralSpec.
func validateCentralSpec(c *dbapi.CentralSpec) error {
	err := validateCoreV1Resources(&c.Resources)
	if err != nil {
		return fmt.Errorf("updating resources within CentralSpec: %w", err)
	}
	return nil
}

// validateScannerSpec validates the ScannerSpec using the non-zero fields from the API's ScannerSpec.
func validateScannerSpec(s *dbapi.ScannerSpec) error {
	var err error
	err = validateCoreV1Resources(&s.Analyzer.Resources)
	if err != nil {
		return fmt.Errorf("updating resources within ScannerSpec Analyzer: %w", err)
	}
	err = validateCoreV1Resources(&s.Db.Resources)
	if err != nil {
		return fmt.Errorf("updating resources within ScannerSpec DB: %w", err)
	}
	return nil
}

func updateCentralRequest(request *dbapi.CentralRequest, strategicPatch []byte) error {

	var patchMap map[string]interface{}
	err := json.Unmarshal(strategicPatch, &patchMap)
	if err != nil {
		return fmt.Errorf("unmarshalling strategic merge patch: %w", err)
	}
	// only keep central and scanner keys
	for k := range patchMap {
		if k != "central" && k != "scanner" {
			delete(patchMap, k)
		}
	}
	patchBytes, err := json.Marshal(patchMap)
	if err != nil {
		return fmt.Errorf("marshalling strategic merge patch: %w", err)
	}

	var centralBytes = "{}"
	if len(request.Central) > 0 {
		centralBytes = string(request.Central)
	}
	var scannerBytes = "{}"
	if len(request.Scanner) > 0 {
		scannerBytes = string(request.Scanner)
	}

	var originalBytes = fmt.Sprintf("{\"central\":%s,\"scanner\":%s,\"forceReconcile\":\"%s\"}", centralBytes, scannerBytes, request.ForceReconcile)

	type Original struct {
		Central        *dbapi.CentralSpec `json:"central,omitempty"`
		Scanner        *dbapi.ScannerSpec `json:"scanner,omitempty"`
		ForceReconcile string             `json:"forceReconcile,omitempty"`
	}

	// apply the patch
	mergedBytes, err := strategicpatch.StrategicMergePatch([]byte(originalBytes), patchBytes, Original{})
	if err != nil {
		return fmt.Errorf("applying strategic merge patch: %w", err)
	}
	var merged Original
	if err := json.Unmarshal(mergedBytes, &merged); err != nil {
		return fmt.Errorf("unmarshalling merged CentralRequest: %w", err)
	}

	if merged.Central == nil {
		merged.Central = &dbapi.CentralSpec{}
	}

	if merged.Scanner == nil {
		merged.Scanner = &dbapi.ScannerSpec{}
	}

	err = validateCentralSpec(merged.Central)
	if err != nil {
		return fmt.Errorf("updating CentralSpec from CentralUpdateRequest: %w", err)
	}
	err = validateScannerSpec(merged.Scanner)
	if err != nil {
		return fmt.Errorf("updating ScannerSpec from CentralUpdateRequest: %w", err)
	}

	newCentralBytes, err := json.Marshal(merged.Central)
	if err != nil {
		return fmt.Errorf("marshalling CentralSpec: %w", err)
	}
	newScannerBytes, err := json.Marshal(merged.Scanner)
	if err != nil {
		return fmt.Errorf("marshalling ScannerSpec: %w", err)
	}

	request.Central = newCentralBytes
	request.Scanner = newScannerBytes
	request.ForceReconcile = merged.ForceReconcile

	return nil

}

// Update a Central instance.
func (h adminCentralHandler) Update(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			centralRequest, svcErr := h.service.Get(ctx, id)
			if svcErr != nil {
				return nil, svcErr
			}

			updateBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, errors.NewWithCause(errors.ErrorBadRequest, err, "Reading request body: %s", err.Error())
			}

			// unmarshal the update into a private.CentralUpdateRequest to ensure that it is well-formed
			if err := json.Unmarshal(updateBytes, &private.CentralUpdateRequest{}); err != nil {
				return nil, errors.NewWithCause(errors.ErrorBadRequest, err, "Unmarshalling request body: %s", err.Error())
			}

			err = updateCentralRequest(centralRequest, updateBytes)
			if err != nil {
				return nil, errors.NewWithCause(errors.ErrorBadRequest, err, "Updating CentralRequest: %s", err.Error())
			}

			svcErr = h.service.VerifyAndUpdateDinosaurAdmin(ctx, centralRequest)
			if svcErr != nil {
				return nil, svcErr
			}
			return presenters.PresentDinosaurRequestAdminEndpoint(centralRequest, h.accountService)
		},
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h adminCentralHandler) SetCentralDefaultVersion(w http.ResponseWriter, r *http.Request) {
	centralDefaultVersion := &private.CentralDefaultVersion{}
	cfg := &handlers.HandlerConfig{
		MarshalInto: centralDefaultVersion,
		Validate: []handlers.Validate{
			ValidateCentralDefaultVersion(centralDefaultVersion),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			if err := h.centralDefaultVersionService.SetDefaultVersion(centralDefaultVersion.Version); err != nil {
				return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Set CentralDefaultVersion requests: %s", err.Error())
			}

			return nil, nil
		},
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h adminCentralHandler) GetCentralDefaultVersion(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			version, err := h.centralDefaultVersionService.GetDefaultVersion()
			if err != nil {
				return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Get CentralDefaultVersion requests: %s", err.Error())
			}
			return &private.CentralDefaultVersion{Version: version}, nil
		},
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

type gitOpsAdminHandler struct{}

var _ AdminCentralHandler = (*gitOpsAdminHandler)(nil)

func (g gitOpsAdminHandler) Create(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (g gitOpsAdminHandler) Get(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (g gitOpsAdminHandler) List(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (g gitOpsAdminHandler) Update(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (g gitOpsAdminHandler) Delete(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (g gitOpsAdminHandler) DbDelete(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (g gitOpsAdminHandler) SetCentralDefaultVersion(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (g gitOpsAdminHandler) GetCentralDefaultVersion(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (g gitOpsAdminHandler) Restore(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
