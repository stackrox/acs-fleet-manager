// Package routes ...
package routes

import (
	"fmt"
	"github.com/stackrox/acs-fleet-manager/openapi"
	"net/http"

	"github.com/goava/di"
	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/gitops"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/handlers"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/routes"
	"github.com/stackrox/acs-fleet-manager/pkg/acl"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	ocm "github.com/stackrox/acs-fleet-manager/pkg/client/ocm/impl"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/features"
	coreHandlers "github.com/stackrox/acs-fleet-manager/pkg/handlers"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
	"github.com/stackrox/acs-fleet-manager/pkg/services/account"
	"github.com/stackrox/acs-fleet-manager/pkg/services/authorization"
)

type options struct {
	di.Inject
	ServerConfig         *server.ServerConfig
	OCMConfig            *ocm.OCMConfig
	ProviderConfig       *config.ProviderConfig
	IAMConfig            *iam.IAMConfig
	CentralRequestConfig *config.CentralRequestConfig

	AMSClient               ocm.AMSClient
	Central                 services.DinosaurService
	ClusterService          services.ClusterService
	CloudProviders          services.CloudProvidersService
	DataPlaneCluster        services.DataPlaneClusterService
	DataPlaneCentralService services.DataPlaneCentralService
	AccountService          account.AccountService
	AuthService             authorization.Authorization
	DB                      *db.ConnectionFactory
	Telemetry               *services.Telemetry

	AccessControlListMiddleware *acl.AccessControlListMiddleware
	AccessControlListConfig     *acl.AccessControlListConfig
	FleetShardAuthZConfig       *auth.FleetShardAuthZConfig
	AdminRoleAuthZConfig        *auth.AdminRoleAuthZConfig

	ManagedCentralPresenter *presenters.ManagedCentralPresenter
	GitopsProvider          gitops.ConfigProvider
}

// NewRouteLoader ...
func NewRouteLoader(s options) environments.RouteLoader {
	return &s
}

// AddRoutes ...
func (s *options) AddRoutes(mainRouter *mux.Router) error {
	basePath := fmt.Sprintf("%s/%s", routes.APIEndpoint, routes.FleetManagementAPIPrefix)
	err := s.buildAPIBaseRouter(mainRouter, basePath)
	if err != nil {
		return err
	}

	return nil
}

func (s *options) buildAPIBaseRouter(mainRouter *mux.Router, basePath string) error {
	centralHandler := handlers.NewDinosaurHandler(s.Central, s.ProviderConfig, s.AuthService, s.Telemetry,
		s.CentralRequestConfig)
	cloudProvidersHandler := handlers.NewCloudProviderHandler(s.CloudProviders, s.ProviderConfig)
	serviceStatusHandler := handlers.NewServiceStatusHandler(s.Central, s.AccessControlListConfig)
	cloudAccountsHandler := handlers.NewCloudAccountsHandler(s.AMSClient)

	authorizeMiddleware := s.AccessControlListMiddleware.Authorize
	requireOrgID := auth.NewRequireOrgIDMiddleware().RequireOrgID(errors.ErrorUnauthenticated)
	requireIssuer := auth.NewRequireIssuerMiddleware().RequireIssuer(
		append(s.IAMConfig.AdditionalSSOIssuers.GetURIs(), s.IAMConfig.RedhatSSORealm.ValidIssuerURI), errors.ErrorUnauthenticated)
	requireTermsAcceptance := auth.NewRequireTermsAcceptanceMiddleware().RequireTermsAcceptance(s.ServerConfig.EnableTermsAcceptance, s.AMSClient, errors.ErrorTermsNotAccepted)

	// base path.
	apiRouter := mainRouter.PathPrefix(basePath).Subrouter()

	// /v1
	apiV1Router := apiRouter.PathPrefix("/v1").Subrouter()

	//  /openapi
	apiV1Router.HandleFunc("/openapi", openapi.HandleGetFleetManagerOpenApiDefinition()).Methods(http.MethodGet)
	
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
	apiV1CentralsRouter.HandleFunc("", centralHandler.List).
		Name(logger.NewLogEvent("list-central", "list all central").ToString()).
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

	apiV1CloudAccountsRouter := apiV1Router.PathPrefix("/cloud_accounts").Subrouter()
	apiV1CloudAccountsRouter.HandleFunc("", cloudAccountsHandler.Get).
		Name(logger.NewLogEvent("get-cloud-accounts", "list all cloud accounts belonging to user org").ToString()).
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
	dataPlaneCentralHandler := handlers.NewDataPlaneDinosaurHandler(s.DataPlaneCentralService, s.Central, s.ManagedCentralPresenter, s.GitopsProvider)
	apiV1DataPlaneRequestsRouter := apiV1Router.PathPrefix(routes.PrivateAPIPrefix).Subrouter()
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}", dataPlaneClusterHandler.GetDataPlaneClusterConfig).
		Name(logger.NewLogEvent("get-dataplane-cluster-config", "get dataplane cluster config by id").ToString()).
		Methods(http.MethodGet)
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/status", dataPlaneClusterHandler.UpdateDataPlaneClusterStatus).
		Name(logger.NewLogEvent("update-dataplane-cluster-status", "update dataplane cluster status by id").ToString()).
		Methods(http.MethodPut)
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/centrals/status", dataPlaneCentralHandler.UpdateDinosaurStatuses).
		Name(logger.NewLogEvent("update-dataplane-centrals-status", "update dataplane centrals status by id").ToString()).
		Methods(http.MethodPut)
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/centrals", dataPlaneCentralHandler.GetAll).
		Name(logger.NewLogEvent("list-dataplane-centrals", "list all dataplane centrals").ToString()).
		Methods(http.MethodGet)

	// /agent-clusters/
	// used for lazy loading additional data not added to the list requests e.g secrets
	apiV1DataPlaneRequestsRouter.HandleFunc("/centrals/{id}", dataPlaneCentralHandler.GetByID).
		Name(logger.NewLogEvent("get-dataplane-central-by-id", "get a single dataplane central").ToString()).
		Methods(http.MethodGet)

	// deliberately returns 404 here if the request doesn't have the required role, so that it will appear as if the endpoint doesn't exist
	auth.UseFleetShardAuthorizationMiddleware(apiV1DataPlaneRequestsRouter, s.IAMConfig, s.FleetShardAuthZConfig)

	adminCentralHandler := handlers.NewAdminCentralHandler(s.Central, s.AccountService, s.ProviderConfig, s.Telemetry)
	adminRouter := apiV1Router.PathPrefix(routes.AdminAPIPrefix).Subrouter()

	adminRouter.Use(auth.NewRequireIssuerMiddleware().RequireIssuer(
		[]string{s.IAMConfig.InternalSSORealm.ValidIssuerURI}, errors.ErrorNotFound))
	adminRouter.Use(auth.NewRolesAuhzMiddleware(s.AdminRoleAuthZConfig).RequireRolesForMethods(errors.ErrorNotFound))
	adminRouter.Use(auth.NewAuditLogMiddleware().AuditLog(errors.ErrorNotFound))
	adminCentralsRouter := adminRouter.PathPrefix("/centrals").Subrouter()

	adminDbCentralsRouter := adminCentralsRouter.PathPrefix("/db").Subrouter()
	adminDbCentralsRouter.HandleFunc("/{id}", adminCentralHandler.DbDelete).
		Name(logger.NewLogEvent("admin-db-delete-central", "[admin] delete central by id").ToString()).
		Methods(http.MethodDelete)

	adminCentralsRouter.HandleFunc("", adminCentralHandler.List).
		Name(logger.NewLogEvent("admin-list-centrals", "[admin] list all centrals").ToString()).
		Methods(http.MethodGet)
	adminCentralsRouter.HandleFunc("/{id}", adminCentralHandler.Get).
		Name(logger.NewLogEvent("admin-get-central", "[admin] get central by id").ToString()).
		Methods(http.MethodGet)
	adminCentralsRouter.HandleFunc("/{id}", adminCentralHandler.Delete).
		Name(logger.NewLogEvent("admin-delete-central", "[admin] delete central by id").ToString()).
		Methods(http.MethodDelete)
	adminCentralsRouter.HandleFunc("/{id}/restore", adminCentralHandler.Restore).
		Name(logger.NewLogEvent("admin-restore-central", "[admin] restore central by id").ToString()).
		Methods(http.MethodPost)
	adminCentralsRouter.HandleFunc("/{id}/rotate-secrets", adminCentralHandler.RotateSecrets).
		Name(logger.NewLogEvent("admin-rotate-central-secrets", "[admin] rotate central secrets by id").ToString()).
		Methods(http.MethodPost)
	adminCentralsRouter.HandleFunc("/{id}/expired-at", adminCentralHandler.PatchExpiredAt).
		Name(logger.NewLogEvent("admin-expired-at", "[admin] set `expired_at` central property").ToString()).
		Methods(http.MethodPatch)
	adminCentralsRouter.HandleFunc("/{id}/name", adminCentralHandler.PatchName).
		Name(logger.NewLogEvent("admin-name", "[admin] set `name` central property").ToString()).
		Methods(http.MethodPatch)
	adminCentralsRouter.HandleFunc("/{id}/billing", adminCentralHandler.PatchBillingParameters).
		Name(logger.NewLogEvent("admin-billing", "[admin] change central billing parameters").ToString()).
		Methods(http.MethodPatch)

	if features.ClusterMigration.Enabled() {
		adminCentralsRouter.HandleFunc("/{id}/assign-cluster", adminCentralHandler.AssignCluster).
			Name(logger.NewLogEvent("admin-central-assign-cluster", "[admin] change central cluster assignment").ToString()).
			Methods(http.MethodPost)
	}

	adminCentralsRouter.HandleFunc("/{id}/traits", adminCentralHandler.ListTraits).
		Name(logger.NewLogEvent("admin-list-traits", "[admin] list central traits").ToString()).
		Methods(http.MethodGet)
	adminCentralsRouter.HandleFunc("/{id}/traits/{trait}", adminCentralHandler.GetTrait).
		Name(logger.NewLogEvent("admin-get-trait", "[admin] check existence of a central trait").ToString()).
		Methods(http.MethodGet)
	adminCentralsRouter.HandleFunc("/{id}/traits/{trait}", adminCentralHandler.AddTrait).
		Name(logger.NewLogEvent("admin-put-trait", "[admin] add a central trait").ToString()).
		Methods(http.MethodPut)
	adminCentralsRouter.HandleFunc("/{id}/traits/{trait}", adminCentralHandler.DeleteTrait).
		Name(logger.NewLogEvent("admin-delete-trait", "[admin] delete central trait").ToString()).
		Methods(http.MethodDelete)

	adminCreateRouter := adminCentralsRouter.NewRoute().Subrouter()
	adminCreateRouter.HandleFunc("", adminCentralHandler.Create).Methods(http.MethodPost)

	return nil
}
