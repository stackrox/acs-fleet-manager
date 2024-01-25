package api

import (
	"context"
	"net/http"

	admin "github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
)

//go:generate moq -rm -out ../mocks/client_moq.go -pkg mocks . PublicAPI PrivateAPI AdminAPI

// PublicAPI is a wrapper interface for the fleetmanager client public API.
type PublicAPI interface {
	CreateCentral(ctx context.Context, async bool, request public.CentralRequestPayload) (public.CentralRequest, *http.Response, error)
	DeleteCentralById(ctx context.Context, id string, async bool) (*http.Response, error)
	GetCentralById(ctx context.Context, id string) (public.CentralRequest, *http.Response, error)
	GetCentrals(ctx context.Context, localVarOptionals *public.GetCentralsOpts) (public.CentralRequestList, *http.Response, error)
}

// PrivateAPI is a wrapper interface for the fleetmanager client private API.
type PrivateAPI interface {
	GetCentral(ctx context.Context, centralID string) (private.ManagedCentral, *http.Response, error)
	GetCentrals(ctx context.Context, id string) (private.ManagedCentralList, *http.Response, error)
	UpdateCentralClusterStatus(ctx context.Context, id string, requestBody map[string]private.DataPlaneCentralStatus) (*http.Response, error)
	UpdateAgentClusterStatus(ctx context.Context, id string, request private.DataPlaneClusterUpdateStatusRequest) (*http.Response, error)
}

// AdminAPI is a wrapper interface for the fleetmanager client admin API.
type AdminAPI interface {
	GetCentrals(ctx context.Context, localVarOptionals *admin.GetCentralsOpts) (admin.CentralList, *http.Response, error)
	CreateCentral(ctx context.Context, async bool, centralRequestPayload admin.CentralRequestPayload) (admin.CentralRequest, *http.Response, error)
	DeleteDbCentralById(ctx context.Context, id string) (*http.Response, error)
	CentralRotateSecrets(ctx context.Context, id string, centralRotateSecretsRequest admin.CentralRotateSecretsRequest) (*http.Response, error)
	UpdateCentralNameById(ctx context.Context, id string, centralUpdateNameRequest admin.CentralUpdateNameRequest) (admin.Central, *http.Response, error)
}
