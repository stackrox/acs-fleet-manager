package routes

import (
	"fmt"
	"net/http"

	"github.com/stackrox/acs-fleet-manager/pkg/services/sso"

	"github.com/stackrox/acs-fleet-manager/pkg/logger"

	"github.com/stackrox/acs-fleet-manager/pkg/services/account"
	"github.com/stackrox/acs-fleet-manager/pkg/services/authorization"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"

	"github.com/goava/di"
	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	pkgerrors "github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/generated"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/handlers"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/routes"
	"github.com/stackrox/acs-fleet-manager/pkg/acl"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	coreHandlers "github.com/stackrox/acs-fleet-manager/pkg/handlers"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

type options struct {
	di.Inject
	ServerConfig   *server.ServerConfig
	OCMConfig      *ocm.OCMConfig
	ProviderConfig *config.ProviderConfig

	AMSClient                ocm.AMSClient
	Dinosaur                 services.DinosaurService
	CloudProviders           services.CloudProvidersService
	Observatorium            services.ObservatoriumService
	IAM                      sso.IAMService
	DataPlaneCluster         services.DataPlaneClusterService
	DataPlaneDinosaurService services.DataPlaneDinosaurService
	AccountService           account.AccountService
	AuthService              authorization.Authorization
	DB                       *db.ConnectionFactory

	AccessControlListMiddleware *acl.AccessControlListMiddleware
	AccessControlListConfig     *acl.AccessControlListConfig
	FleetShardAuthZConfig       *auth.FleetShardAuthZConfig
}

// NewRouteLoader ...
func NewRouteLoader(s options) environments.RouteLoader {
	return &s
}

// AddRoutes ...
func (s *options) AddRoutes(mainRouter *mux.Router) error {
	basePath := fmt.Sprintf("%s/%s", routes.APIEndpoint, routes.DinosaursFleetManagementAPIPrefix)
	err := s.buildAPIBaseRouter(mainRouter, basePath, "fleet-manager.yaml")
	if err != nil {
		return err
	}

	return nil
}

func (s *options) buildAPIBaseRouter(mainRouter *mux.Router, basePath string, openAPIFilePath string) error {
	openAPIDefinitions, err := shared.LoadOpenAPISpec(generated.Asset, openAPIFilePath)
	if err != nil {
		return pkgerrors.Wrapf(err, "can't load OpenAPI specification")
	}

	dinosaurHandler := handlers.NewDinosaurHandler(s.Dinosaur, s.ProviderConfig, s.AuthService)
	cloudProvidersHandler := handlers.NewCloudProviderHandler(s.CloudProviders, s.ProviderConfig)
	errorsHandler := coreHandlers.NewErrorsHandler()
	metricsHandler := handlers.NewMetricsHandler(s.Observatorium)
	serviceStatusHandler := handlers.NewServiceStatusHandler(s.Dinosaur, s.AccessControlListConfig)

	authorizeMiddleware := s.AccessControlListMiddleware.Authorize
	requireOrgID := auth.NewRequireOrgIDMiddleware().RequireOrgID(errors.ErrorUnauthenticated)
	requireIssuer := auth.NewRequireIssuerMiddleware().RequireIssuer(
		append(s.IAM.GetConfig().AdditionalSSOIssuers.URIs, s.ServerConfig.TokenIssuerURL), errors.ErrorUnauthenticated)
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
	apiV1DinosaursRouter := apiV1Router.PathPrefix("/centrals").Subrouter()
	apiV1DinosaursRouter.HandleFunc("/{id}", dinosaurHandler.Get).
		Name(logger.NewLogEvent("get-central", "get a central instance").ToString()).
		Methods(http.MethodGet)
	apiV1DinosaursRouter.HandleFunc("/{id}", dinosaurHandler.Delete).
		Name(logger.NewLogEvent("delete-central", "delete a central instance").ToString()).
		Methods(http.MethodDelete)
	apiV1DinosaursRouter.HandleFunc("/{id}", dinosaurHandler.Update).
		Name(logger.NewLogEvent("update-central", "update a central instance").ToString()).
		Methods(http.MethodPatch)
	apiV1DinosaursRouter.HandleFunc("", dinosaurHandler.List).
		Name(logger.NewLogEvent("list-central", "list all central").ToString()).
		Methods(http.MethodGet)
	apiV1DinosaursRouter.Use(requireIssuer)
	apiV1DinosaursRouter.Use(requireOrgID)
	apiV1DinosaursRouter.Use(authorizeMiddleware)

	apiV1DinosaursCreateRouter := apiV1DinosaursRouter.NewRoute().Subrouter()
	apiV1DinosaursCreateRouter.HandleFunc("", dinosaurHandler.Create).Methods(http.MethodPost)
	apiV1DinosaursCreateRouter.Use(requireTermsAcceptance)

	//  /dinosaurs/{id}/metrics
	apiV1MetricsRouter := apiV1DinosaursRouter.PathPrefix("/{id}/metrics").Subrouter()
	apiV1MetricsRouter.HandleFunc("/query_range", metricsHandler.GetMetricsByRangeQuery).
		Name(logger.NewLogEvent("get-metrics", "list metrics by range").ToString()).
		Methods(http.MethodGet)
	apiV1MetricsRouter.HandleFunc("/query", metricsHandler.GetMetricsByInstantQuery).
		Name(logger.NewLogEvent("get-metrics-instant", "get metrics by instant").ToString()).
		Methods(http.MethodGet)

	// /centrals/{id}/metrics/federate
	// federate endpoint separated from the rest of the /centrals endpoints as it needs to support auth from both sso.redhat.com and mas-sso
	// NOTE: this is only a temporary solution. MAS SSO auth support should be removed once we migrate to sso.redhat.com (TODO: to be done as part of MGDSTRM-6159)
	apiV1MetricsFederateRouter := apiV1Router.PathPrefix("/centrals/{id}/metrics/federate").Subrouter()
	apiV1MetricsFederateRouter.HandleFunc("", metricsHandler.FederateMetrics).
		Name(logger.NewLogEvent("get-federate-metrics", "get federate metrics by id").ToString()).
		Methods(http.MethodGet)
	apiV1MetricsFederateRouter.Use(auth.NewRequireIssuerMiddleware().RequireIssuer(
		append(s.IAM.GetConfig().AdditionalSSOIssuers.URIs, s.ServerConfig.TokenIssuerURL,
			s.IAM.GetConfig().RedhatSSORealm.ValidIssuerURI), errors.ErrorUnauthenticated))
	apiV1MetricsFederateRouter.Use(requireOrgID)
	apiV1MetricsFederateRouter.Use(authorizeMiddleware)

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

	// /agent-clusters/{id}
	dataPlaneClusterHandler := handlers.NewDataPlaneClusterHandler(s.DataPlaneCluster)
	dataPlaneDinosaurHandler := handlers.NewDataPlaneDinosaurHandler(s.DataPlaneDinosaurService, s.Dinosaur)
	apiV1DataPlaneRequestsRouter := apiV1Router.PathPrefix("/agent-clusters").Subrouter()
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}", dataPlaneClusterHandler.GetDataPlaneClusterConfig).
		Name(logger.NewLogEvent("get-dataplane-cluster-config", "get dataplane cluster config by id").ToString()).
		Methods(http.MethodGet)
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/status", dataPlaneClusterHandler.UpdateDataPlaneClusterStatus).
		Name(logger.NewLogEvent("update-dataplane-cluster-status", "update dataplane cluster status by id").ToString()).
		Methods(http.MethodPut)
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/centrals/status", dataPlaneDinosaurHandler.UpdateDinosaurStatuses).
		Name(logger.NewLogEvent("update-dataplane-centrals-status", "update dataplane centrals status by id").ToString()).
		Methods(http.MethodPut)
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/centrals", dataPlaneDinosaurHandler.GetAll).
		Name(logger.NewLogEvent("list-dataplane-centrals", "list all dataplane centrals").ToString()).
		Methods(http.MethodGet)
	// deliberately returns 404 here if the request doesn't have the required role, so that it will appear as if the endpoint doesn't exist
	auth.UseFleetShardAuthorizationMiddleware(apiV1DataPlaneRequestsRouter,
		s.IAM.GetConfig().RedhatSSORealm.ValidIssuerURI, s.FleetShardAuthZConfig)

	adminDinosaurHandler := handlers.NewAdminDinosaurHandler(s.Dinosaur, s.AccountService, s.ProviderConfig)
	adminRouter := apiV1Router.PathPrefix("/admin").Subrouter()
	rolesMapping := map[string][]string{
		http.MethodGet:    {auth.FleetManagerAdminReadRole, auth.FleetManagerAdminWriteRole, auth.FleetManagerAdminFullRole},
		http.MethodPatch:  {auth.FleetManagerAdminWriteRole, auth.FleetManagerAdminFullRole},
		http.MethodDelete: {auth.FleetManagerAdminFullRole},
	}

	// TODO(ROX-11683): For now using RH SSO issuer for the admin API, but needs to be re-visited within this ticket.
	adminRouter.Use(auth.NewRequireIssuerMiddleware().RequireIssuer(
		[]string{s.IAM.GetConfig().RedhatSSORealm.ValidIssuerURI}, errors.ErrorNotFound))
	adminRouter.Use(auth.NewRolesAuhzMiddleware().RequireRolesForMethods(rolesMapping, errors.ErrorNotFound))
	adminRouter.Use(auth.NewAuditLogMiddleware().AuditLog(errors.ErrorNotFound))
	adminRouter.HandleFunc("/dinosaurs", adminDinosaurHandler.List).
		Name(logger.NewLogEvent("admin-list-dinosaurs", "[admin] list all dinosaurs").ToString()).
		Methods(http.MethodGet)
	adminRouter.HandleFunc("/dinosaurs/{id}", adminDinosaurHandler.Get).
		Name(logger.NewLogEvent("admin-get-dinosaur", "[admin] get dinosaur by id").ToString()).
		Methods(http.MethodGet)
	adminRouter.HandleFunc("/dinosaurs/{id}", adminDinosaurHandler.Delete).
		Name(logger.NewLogEvent("admin-delete-dinosaur", "[admin] delete dinosaur by id").ToString()).
		Methods(http.MethodDelete)
	adminRouter.HandleFunc("/dinosaurs/{id}", adminDinosaurHandler.Update).
		Name(logger.NewLogEvent("admin-update-dinosaur", "[admin] update dinosaur by id").ToString()).
		Methods(http.MethodPatch)

	return nil
}
