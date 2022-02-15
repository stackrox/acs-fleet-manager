package handlers

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services/authorization"

	"github.com/gorilla/mux"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services"
)

type dinosaurHandler struct {
	service        services.DinosaurService
	providerConfig *config.ProviderConfig
	authService    authorization.Authorization
}

func NewDinosaurHandler(service services.DinosaurService, providerConfig *config.ProviderConfig, authService authorization.Authorization) *dinosaurHandler {
	return &dinosaurHandler{
		service:        service,
		providerConfig: providerConfig,
		authService:    authService,
	}
}

func (h dinosaurHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dinosaurRequest public.DinosaurRequestPayload
	ctx := r.Context()
	convDinosaur := &dbapi.DinosaurRequest{}

	cfg := &handlers.HandlerConfig{
		MarshalInto: &dinosaurRequest,
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "creating dinosaur requests"),
			handlers.ValidateLength(&dinosaurRequest.Name, "name", &handlers.MinRequiredFieldLength, &MaxDinosaurNameLength),
			ValidDinosaurClusterName(&dinosaurRequest.Name, "name"),
			ValidateDinosaurClusterNameIsUnique(&dinosaurRequest.Name, h.service, r.Context()),
			ValidateDinosaurClaims(ctx, &dinosaurRequest, convDinosaur),
			ValidateCloudProvider(&h.service, convDinosaur, h.providerConfig, "creating dinosaur requests"),
			handlers.ValidateMultiAZEnabled(&dinosaurRequest.MultiAz, "creating dinosaur requests"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			svcErr := h.service.RegisterDinosaurJob(convDinosaur)
			if svcErr != nil {
				return nil, svcErr
			}
			return presenters.PresentDinosaurRequest(convDinosaur), nil
		},
	}

	// return 202 status accepted
	handlers.Handle(w, r, cfg, http.StatusAccepted)
}

func (h dinosaurHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			dinosaurRequest, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			return presenters.PresentDinosaurRequest(dinosaurRequest), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

// Delete is the handler for deleting a dinosaur request
func (h dinosaurHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

func (h dinosaurHandler) List(w http.ResponseWriter, r *http.Request) {
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

			dinosaurRequestList := public.DinosaurRequestList{
				Kind:  "DinosaurRequestList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []public.DinosaurRequest{},
			}

			for _, dinosaurRequest := range dinosaurRequests {
				converted := presenters.PresentDinosaurRequest(dinosaurRequest)
				dinosaurRequestList.Items = append(dinosaurRequestList.Items, converted)
			}

			return dinosaurRequestList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

// Update is the handler for updating a dinosaur request
func (h dinosaurHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dinosaurUpdateReq public.DinosaurUpdateRequest
	id := mux.Vars(r)["id"]
	ctx := r.Context()
	dinosaurRequest, dinosaurGetError := h.service.Get(ctx, id)
	validateDinosaurFound := func() handlers.Validate {
		return func() *errors.ServiceError {
			return dinosaurGetError
		}
	}
	cfg := &handlers.HandlerConfig{
		MarshalInto: &dinosaurUpdateReq,
		Validate: []handlers.Validate{
			validateDinosaurFound(),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			// TODO implement update logic
			var updDinosaurRequest *dbapi.DinosaurRequest
			svcErr := h.service.Update(updDinosaurRequest)
			if svcErr != nil {
				return nil, svcErr
			}

			return presenters.PresentDinosaurRequest(dinosaurRequest), nil
		},
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}
