package routes

import (
	"fmt"
	"net/http"

	"github.com/stackrox/acs-fleet-manager/pkg/logger"

	"github.com/stackrox/acs-fleet-manager/pkg/services/account"
	"github.com/stackrox/acs-fleet-manager/pkg/services/authorization"

	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/config"

	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/generated"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/handlers"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/services"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/routes"
	"github.com/stackrox/acs-fleet-manager/pkg/acl"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	coreHandlers "github.com/stackrox/acs-fleet-manager/pkg/handlers"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
	coreServices "github.com/stackrox/acs-fleet-manager/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"github.com/goava/di"
	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	pkgerrors "github.com/pkg/errors"
)

type options struct {
	di.Inject
	ServerConfig   *server.ServerConfig
	OCMConfig      *ocm.OCMConfig
	ProviderConfig *config.ProviderConfig

	AMSClient                ocm.AMSClient
	CentralService                 services.CentralService
	CloudProviders           services.CloudProvidersService
	// FIXME Observatorium            services.ObservatoriumService
	Keycloak                 coreServices.DinosaurKeycloakService
	// FIXME DataPlaneCluster         services.DataPlaneClusterService
	// FIXME DataPlaneDinosaurService services.DataPlaneDinosaurService
	AccountService           account.AccountService
	AuthService              authorization.Authorization
	DB                       *db.ConnectionFactory

	AccessControlListMiddleware *acl.AccessControlListMiddleware
	AccessControlListConfig     *acl.AccessControlListConfig
}

func NewRouteLoader(s options) environments.RouteLoader {
	return &s
}

func (s *options) AddRoutes(mainRouter *mux.Router) error {
	basePath := fmt.Sprintf("%s/%s", routes.ApiEndpoint, routes.FleetManagementApiPrefix)
	err := s.buildApiBaseRouter(mainRouter, basePath, "fleet-manager.yaml")
	if err != nil {
		return err
	}

	return nil
}

func (s *options) buildApiBaseRouter(mainRouter *mux.Router, basePath string, openApiFilePath string) error {
	openAPIDefinitions, err := shared.LoadOpenAPISpec(generated.Asset, openApiFilePath)
	if err != nil {
		return pkgerrors.Wrapf(err, "can't load OpenAPI specification")
	}

	centralHandler := handlers.NewCentralHandler(s.CentralService, s.ProviderConfig, s.AuthService)
	serviceStatusHandler := handlers.NewServiceStatusHandler(s.CentralService, s.AccessControlListConfig)
	cloudProvidersHandler := handlers.NewCloudProviderHandler(s.CloudProviders, s.ProviderConfig)
	errorsHandler := coreHandlers.NewErrorsHandler()
	
	authorizeMiddleware := s.AccessControlListMiddleware.Authorize
	requireOrgID := auth.NewRequireOrgIDMiddleware().RequireOrgID(errors.ErrorUnauthenticated)
	requireIssuer := auth.NewRequireIssuerMiddleware().RequireIssuer([]string{s.ServerConfig.TokenIssuerURL}, errors.ErrorUnauthenticated)
	requireTermsAcceptance := auth.NewRequireTermsAcceptanceMiddleware().RequireTermsAcceptance(s.ServerConfig.EnableTermsAcceptance, s.AMSClient, errors.ErrorTermsNotAccepted)

	// base path.
	apiRouter := mainRouter.PathPrefix(basePath).Subrouter()

	// /v1
	apiV1Router := apiRouter.PathPrefix("/v1").Subrouter()

	//  /openapi
	apiV1Router.HandleFunc("/openapi", coreHandlers.NewOpenAPIHandler(openAPIDefinitions).Get).Methods(http.MethodGet)

	//  /errors
	apiV1ErrorsRouter := apiV1Router.PathPrefix("/errors").Subrouter()
	apiV1ErrorsRouter.HandleFunc("", errorsHandler.List).Methods(http.MethodGet)
	apiV1ErrorsRouter.HandleFunc("/{id}", errorsHandler.Get).Methods(http.MethodGet)

	// /status
	apiV1Status := apiV1Router.PathPrefix("/status").Subrouter()
	apiV1Status.HandleFunc("", serviceStatusHandler.Get).Methods(http.MethodGet)
	apiV1Status.Use(requireIssuer)

	v1Collections := []api.CollectionMetadata{}

	//  /centrals
	v1Collections = append(v1Collections, api.CollectionMetadata{
		ID:   "centrals",
		Kind: "CentralList",
	})
	apiV1CentralsRouter := apiV1Router.PathPrefix("/centrals").Subrouter()
	apiV1CentralsRouter.HandleFunc("/{id}", centralHandler.Get).
		Name(logger.NewLogEvent("get-central", "get a central instance").ToString()).
		Methods(http.MethodGet)
	apiV1CentralsRouter.HandleFunc("/{id}", centralHandler.Delete).
		Name(logger.NewLogEvent("delete-central", "delete a central instance").ToString()).
		Methods(http.MethodDelete)
	apiV1CentralsRouter.HandleFunc("/{id}", centralHandler.Update).
		Name(logger.NewLogEvent("update-central", "update a central instance").ToString()).
		Methods(http.MethodPatch)
	apiV1CentralsRouter.HandleFunc("", centralHandler.List).
		Name(logger.NewLogEvent("list-central", "list all centrals").ToString()).
		Methods(http.MethodGet)
	apiV1CentralsRouter.Use(requireIssuer)
	apiV1CentralsRouter.Use(requireOrgID)
	apiV1CentralsRouter.Use(authorizeMiddleware)

	apiV1CentralsCreateRouter := apiV1CentralsRouter.NewRoute().Subrouter()
	apiV1CentralsCreateRouter.HandleFunc("", centralHandler.Create).Methods(http.MethodPost)
	apiV1CentralsCreateRouter.Use(requireTermsAcceptance)

	//  /cloud_providers
	v1Collections = append(v1Collections, api.CollectionMetadata{
		ID:   "cloud_providers",
		Kind: "CloudProviderList",
	})
	apiV1CloudProvidersRouter := apiV1Router.PathPrefix("/cloud_providers").Subrouter()
	apiV1CloudProvidersRouter.HandleFunc("", cloudProvidersHandler.ListCloudProviders).
		Name(logger.NewLogEvent("list-cloud-providers", "list all cloud providers").ToString()).
		Methods(http.MethodGet)
	apiV1CloudProvidersRouter.HandleFunc("/{id}/regions", cloudProvidersHandler.ListCloudProviderRegions).
		Name(logger.NewLogEvent("list-regions", "list cloud provider regions").ToString()).
		Methods(http.MethodGet)

	v1Metadata := api.VersionMetadata{
		ID:          "v1",
		Collections: v1Collections,
	}
	apiMetadata := api.Metadata{
		ID: "rhacs",
		Versions: []api.VersionMetadata{
			v1Metadata,
		},
	}
	apiRouter.HandleFunc("", apiMetadata.ServeHTTP).Methods(http.MethodGet)
	apiRouter.Use(coreHandlers.MetricsMiddleware)
	apiRouter.Use(db.TransactionMiddleware(s.DB))
	apiRouter.Use(gorillaHandlers.CompressHandler)

	apiV1Router.HandleFunc("", v1Metadata.ServeHTTP).Methods(http.MethodGet)

	return nil
}

// TODO complete following internal/dinosaur/internal/routes/route_loader.go