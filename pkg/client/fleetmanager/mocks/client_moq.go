// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mocks

import (
	"context"
	admin "github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"net/http"
	"sync"
)

// Ensure, that PublicAPIMock does implement fleetmanager.PublicAPI.
// If this is not the case, regenerate this file with moq.
var _ fleetmanager.PublicAPI = &PublicAPIMock{}

// PublicAPIMock is a mock implementation of fleetmanager.PublicAPI.
//
//	func TestSomethingThatUsesPublicAPI(t *testing.T) {
//
//		// make and configure a mocked fleetmanager.PublicAPI
//		mockedPublicAPI := &PublicAPIMock{
//			CreateCentralFunc: func(ctx context.Context, async bool, request public.CentralRequestPayload) (public.CentralRequest, *http.Response, error) {
//				panic("mock out the CreateCentral method")
//			},
//			DeleteCentralByIdFunc: func(ctx context.Context, id string, async bool) (*http.Response, error) {
//				panic("mock out the DeleteCentralById method")
//			},
//			GetCentralByIdFunc: func(ctx context.Context, id string) (public.CentralRequest, *http.Response, error) {
//				panic("mock out the GetCentralById method")
//			},
//			GetCentralsFunc: func(ctx context.Context, localVarOptionals *public.GetCentralsOpts) (public.CentralRequestList, *http.Response, error) {
//				panic("mock out the GetCentrals method")
//			},
//			GetCloudProviderRegionsFunc: func(ctx context.Context, id string, localVarOptionals *public.GetCloudProviderRegionsOpts) (public.CloudRegionList, *http.Response, error) {
//				panic("mock out the GetCloudProviderRegions method")
//			},
//			GetCloudProvidersFunc: func(ctx context.Context, localVarOptionals *public.GetCloudProvidersOpts) (public.CloudProviderList, *http.Response, error) {
//				panic("mock out the GetCloudProviders method")
//			},
//		}
//
//		// use mockedPublicAPI in code that requires fleetmanager.PublicAPI
//		// and then make assertions.
//
//	}
type PublicAPIMock struct {
	// CreateCentralFunc mocks the CreateCentral method.
	CreateCentralFunc func(ctx context.Context, async bool, request public.CentralRequestPayload) (public.CentralRequest, *http.Response, error)

	// DeleteCentralByIdFunc mocks the DeleteCentralById method.
	DeleteCentralByIdFunc func(ctx context.Context, id string, async bool) (*http.Response, error)

	// GetCentralByIdFunc mocks the GetCentralById method.
	GetCentralByIdFunc func(ctx context.Context, id string) (public.CentralRequest, *http.Response, error)

	// GetCentralsFunc mocks the GetCentrals method.
	GetCentralsFunc func(ctx context.Context, localVarOptionals *public.GetCentralsOpts) (public.CentralRequestList, *http.Response, error)

	// GetCloudProviderRegionsFunc mocks the GetCloudProviderRegions method.
	GetCloudProviderRegionsFunc func(ctx context.Context, id string, localVarOptionals *public.GetCloudProviderRegionsOpts) (public.CloudRegionList, *http.Response, error)

	// GetCloudProvidersFunc mocks the GetCloudProviders method.
	GetCloudProvidersFunc func(ctx context.Context, localVarOptionals *public.GetCloudProvidersOpts) (public.CloudProviderList, *http.Response, error)

	// calls tracks calls to the methods.
	calls struct {
		// CreateCentral holds details about calls to the CreateCentral method.
		CreateCentral []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Async is the async argument value.
			Async bool
			// Request is the request argument value.
			Request public.CentralRequestPayload
		}
		// DeleteCentralById holds details about calls to the DeleteCentralById method.
		DeleteCentralById []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// Async is the async argument value.
			Async bool
		}
		// GetCentralById holds details about calls to the GetCentralById method.
		GetCentralById []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
		}
		// GetCentrals holds details about calls to the GetCentrals method.
		GetCentrals []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// LocalVarOptionals is the localVarOptionals argument value.
			LocalVarOptionals *public.GetCentralsOpts
		}
		// GetCloudProviderRegions holds details about calls to the GetCloudProviderRegions method.
		GetCloudProviderRegions []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// LocalVarOptionals is the localVarOptionals argument value.
			LocalVarOptionals *public.GetCloudProviderRegionsOpts
		}
		// GetCloudProviders holds details about calls to the GetCloudProviders method.
		GetCloudProviders []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// LocalVarOptionals is the localVarOptionals argument value.
			LocalVarOptionals *public.GetCloudProvidersOpts
		}
	}
	lockCreateCentral           sync.RWMutex
	lockDeleteCentralById       sync.RWMutex
	lockGetCentralById          sync.RWMutex
	lockGetCentrals             sync.RWMutex
	lockGetCloudProviderRegions sync.RWMutex
	lockGetCloudProviders       sync.RWMutex
}

// CreateCentral calls CreateCentralFunc.
func (mock *PublicAPIMock) CreateCentral(ctx context.Context, async bool, request public.CentralRequestPayload) (public.CentralRequest, *http.Response, error) {
	if mock.CreateCentralFunc == nil {
		panic("PublicAPIMock.CreateCentralFunc: method is nil but PublicAPI.CreateCentral was just called")
	}
	callInfo := struct {
		Ctx     context.Context
		Async   bool
		Request public.CentralRequestPayload
	}{
		Ctx:     ctx,
		Async:   async,
		Request: request,
	}
	mock.lockCreateCentral.Lock()
	mock.calls.CreateCentral = append(mock.calls.CreateCentral, callInfo)
	mock.lockCreateCentral.Unlock()
	return mock.CreateCentralFunc(ctx, async, request)
}

// CreateCentralCalls gets all the calls that were made to CreateCentral.
// Check the length with:
//
//	len(mockedPublicAPI.CreateCentralCalls())
func (mock *PublicAPIMock) CreateCentralCalls() []struct {
	Ctx     context.Context
	Async   bool
	Request public.CentralRequestPayload
} {
	var calls []struct {
		Ctx     context.Context
		Async   bool
		Request public.CentralRequestPayload
	}
	mock.lockCreateCentral.RLock()
	calls = mock.calls.CreateCentral
	mock.lockCreateCentral.RUnlock()
	return calls
}

// DeleteCentralById calls DeleteCentralByIdFunc.
func (mock *PublicAPIMock) DeleteCentralById(ctx context.Context, id string, async bool) (*http.Response, error) {
	if mock.DeleteCentralByIdFunc == nil {
		panic("PublicAPIMock.DeleteCentralByIdFunc: method is nil but PublicAPI.DeleteCentralById was just called")
	}
	callInfo := struct {
		Ctx   context.Context
		ID    string
		Async bool
	}{
		Ctx:   ctx,
		ID:    id,
		Async: async,
	}
	mock.lockDeleteCentralById.Lock()
	mock.calls.DeleteCentralById = append(mock.calls.DeleteCentralById, callInfo)
	mock.lockDeleteCentralById.Unlock()
	return mock.DeleteCentralByIdFunc(ctx, id, async)
}

// DeleteCentralByIdCalls gets all the calls that were made to DeleteCentralById.
// Check the length with:
//
//	len(mockedPublicAPI.DeleteCentralByIdCalls())
func (mock *PublicAPIMock) DeleteCentralByIdCalls() []struct {
	Ctx   context.Context
	ID    string
	Async bool
} {
	var calls []struct {
		Ctx   context.Context
		ID    string
		Async bool
	}
	mock.lockDeleteCentralById.RLock()
	calls = mock.calls.DeleteCentralById
	mock.lockDeleteCentralById.RUnlock()
	return calls
}

// GetCentralById calls GetCentralByIdFunc.
func (mock *PublicAPIMock) GetCentralById(ctx context.Context, id string) (public.CentralRequest, *http.Response, error) {
	if mock.GetCentralByIdFunc == nil {
		panic("PublicAPIMock.GetCentralByIdFunc: method is nil but PublicAPI.GetCentralById was just called")
	}
	callInfo := struct {
		Ctx context.Context
		ID  string
	}{
		Ctx: ctx,
		ID:  id,
	}
	mock.lockGetCentralById.Lock()
	mock.calls.GetCentralById = append(mock.calls.GetCentralById, callInfo)
	mock.lockGetCentralById.Unlock()
	return mock.GetCentralByIdFunc(ctx, id)
}

// GetCentralByIdCalls gets all the calls that were made to GetCentralById.
// Check the length with:
//
//	len(mockedPublicAPI.GetCentralByIdCalls())
func (mock *PublicAPIMock) GetCentralByIdCalls() []struct {
	Ctx context.Context
	ID  string
} {
	var calls []struct {
		Ctx context.Context
		ID  string
	}
	mock.lockGetCentralById.RLock()
	calls = mock.calls.GetCentralById
	mock.lockGetCentralById.RUnlock()
	return calls
}

// GetCentrals calls GetCentralsFunc.
func (mock *PublicAPIMock) GetCentrals(ctx context.Context, localVarOptionals *public.GetCentralsOpts) (public.CentralRequestList, *http.Response, error) {
	if mock.GetCentralsFunc == nil {
		panic("PublicAPIMock.GetCentralsFunc: method is nil but PublicAPI.GetCentrals was just called")
	}
	callInfo := struct {
		Ctx               context.Context
		LocalVarOptionals *public.GetCentralsOpts
	}{
		Ctx:               ctx,
		LocalVarOptionals: localVarOptionals,
	}
	mock.lockGetCentrals.Lock()
	mock.calls.GetCentrals = append(mock.calls.GetCentrals, callInfo)
	mock.lockGetCentrals.Unlock()
	return mock.GetCentralsFunc(ctx, localVarOptionals)
}

// GetCentralsCalls gets all the calls that were made to GetCentrals.
// Check the length with:
//
//	len(mockedPublicAPI.GetCentralsCalls())
func (mock *PublicAPIMock) GetCentralsCalls() []struct {
	Ctx               context.Context
	LocalVarOptionals *public.GetCentralsOpts
} {
	var calls []struct {
		Ctx               context.Context
		LocalVarOptionals *public.GetCentralsOpts
	}
	mock.lockGetCentrals.RLock()
	calls = mock.calls.GetCentrals
	mock.lockGetCentrals.RUnlock()
	return calls
}

// GetCloudProviderRegions calls GetCloudProviderRegionsFunc.
func (mock *PublicAPIMock) GetCloudProviderRegions(ctx context.Context, id string, localVarOptionals *public.GetCloudProviderRegionsOpts) (public.CloudRegionList, *http.Response, error) {
	if mock.GetCloudProviderRegionsFunc == nil {
		panic("PublicAPIMock.GetCloudProviderRegionsFunc: method is nil but PublicAPI.GetCloudProviderRegions was just called")
	}
	callInfo := struct {
		Ctx               context.Context
		ID                string
		LocalVarOptionals *public.GetCloudProviderRegionsOpts
	}{
		Ctx:               ctx,
		ID:                id,
		LocalVarOptionals: localVarOptionals,
	}
	mock.lockGetCloudProviderRegions.Lock()
	mock.calls.GetCloudProviderRegions = append(mock.calls.GetCloudProviderRegions, callInfo)
	mock.lockGetCloudProviderRegions.Unlock()
	return mock.GetCloudProviderRegionsFunc(ctx, id, localVarOptionals)
}

// GetCloudProviderRegionsCalls gets all the calls that were made to GetCloudProviderRegions.
// Check the length with:
//
//	len(mockedPublicAPI.GetCloudProviderRegionsCalls())
func (mock *PublicAPIMock) GetCloudProviderRegionsCalls() []struct {
	Ctx               context.Context
	ID                string
	LocalVarOptionals *public.GetCloudProviderRegionsOpts
} {
	var calls []struct {
		Ctx               context.Context
		ID                string
		LocalVarOptionals *public.GetCloudProviderRegionsOpts
	}
	mock.lockGetCloudProviderRegions.RLock()
	calls = mock.calls.GetCloudProviderRegions
	mock.lockGetCloudProviderRegions.RUnlock()
	return calls
}

// GetCloudProviders calls GetCloudProvidersFunc.
func (mock *PublicAPIMock) GetCloudProviders(ctx context.Context, localVarOptionals *public.GetCloudProvidersOpts) (public.CloudProviderList, *http.Response, error) {
	if mock.GetCloudProvidersFunc == nil {
		panic("PublicAPIMock.GetCloudProvidersFunc: method is nil but PublicAPI.GetCloudProviders was just called")
	}
	callInfo := struct {
		Ctx               context.Context
		LocalVarOptionals *public.GetCloudProvidersOpts
	}{
		Ctx:               ctx,
		LocalVarOptionals: localVarOptionals,
	}
	mock.lockGetCloudProviders.Lock()
	mock.calls.GetCloudProviders = append(mock.calls.GetCloudProviders, callInfo)
	mock.lockGetCloudProviders.Unlock()
	return mock.GetCloudProvidersFunc(ctx, localVarOptionals)
}

// GetCloudProvidersCalls gets all the calls that were made to GetCloudProviders.
// Check the length with:
//
//	len(mockedPublicAPI.GetCloudProvidersCalls())
func (mock *PublicAPIMock) GetCloudProvidersCalls() []struct {
	Ctx               context.Context
	LocalVarOptionals *public.GetCloudProvidersOpts
} {
	var calls []struct {
		Ctx               context.Context
		LocalVarOptionals *public.GetCloudProvidersOpts
	}
	mock.lockGetCloudProviders.RLock()
	calls = mock.calls.GetCloudProviders
	mock.lockGetCloudProviders.RUnlock()
	return calls
}

// Ensure, that PrivateAPIMock does implement fleetmanager.PrivateAPI.
// If this is not the case, regenerate this file with moq.
var _ fleetmanager.PrivateAPI = &PrivateAPIMock{}

// PrivateAPIMock is a mock implementation of fleetmanager.PrivateAPI.
//
//	func TestSomethingThatUsesPrivateAPI(t *testing.T) {
//
//		// make and configure a mocked fleetmanager.PrivateAPI
//		mockedPrivateAPI := &PrivateAPIMock{
//			GetCentralFunc: func(ctx context.Context, centralID string) (private.ManagedCentral, *http.Response, error) {
//				panic("mock out the GetCentral method")
//			},
//			GetCentralsFunc: func(ctx context.Context, id string) (private.ManagedCentralList, *http.Response, error) {
//				panic("mock out the GetCentrals method")
//			},
//			UpdateAgentClusterStatusFunc: func(ctx context.Context, id string, request private.DataPlaneClusterUpdateStatusRequest) (*http.Response, error) {
//				panic("mock out the UpdateAgentClusterStatus method")
//			},
//			UpdateCentralClusterStatusFunc: func(ctx context.Context, id string, requestBody map[string]private.DataPlaneCentralStatus) (*http.Response, error) {
//				panic("mock out the UpdateCentralClusterStatus method")
//			},
//		}
//
//		// use mockedPrivateAPI in code that requires fleetmanager.PrivateAPI
//		// and then make assertions.
//
//	}
type PrivateAPIMock struct {
	// GetCentralFunc mocks the GetCentral method.
	GetCentralFunc func(ctx context.Context, centralID string) (private.ManagedCentral, *http.Response, error)

	// GetCentralsFunc mocks the GetCentrals method.
	GetCentralsFunc func(ctx context.Context, id string) (private.ManagedCentralList, *http.Response, error)

	// UpdateAgentClusterStatusFunc mocks the UpdateAgentClusterStatus method.
	UpdateAgentClusterStatusFunc func(ctx context.Context, id string, request private.DataPlaneClusterUpdateStatusRequest) (*http.Response, error)

	// UpdateCentralClusterStatusFunc mocks the UpdateCentralClusterStatus method.
	UpdateCentralClusterStatusFunc func(ctx context.Context, id string, requestBody map[string]private.DataPlaneCentralStatus) (*http.Response, error)

	// calls tracks calls to the methods.
	calls struct {
		// GetCentral holds details about calls to the GetCentral method.
		GetCentral []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// CentralID is the centralID argument value.
			CentralID string
		}
		// GetCentrals holds details about calls to the GetCentrals method.
		GetCentrals []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
		}
		// UpdateAgentClusterStatus holds details about calls to the UpdateAgentClusterStatus method.
		UpdateAgentClusterStatus []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// Request is the request argument value.
			Request private.DataPlaneClusterUpdateStatusRequest
		}
		// UpdateCentralClusterStatus holds details about calls to the UpdateCentralClusterStatus method.
		UpdateCentralClusterStatus []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// RequestBody is the requestBody argument value.
			RequestBody map[string]private.DataPlaneCentralStatus
		}
	}
	lockGetCentral                 sync.RWMutex
	lockGetCentrals                sync.RWMutex
	lockUpdateAgentClusterStatus   sync.RWMutex
	lockUpdateCentralClusterStatus sync.RWMutex
}

// GetCentral calls GetCentralFunc.
func (mock *PrivateAPIMock) GetCentral(ctx context.Context, centralID string) (private.ManagedCentral, *http.Response, error) {
	if mock.GetCentralFunc == nil {
		panic("PrivateAPIMock.GetCentralFunc: method is nil but PrivateAPI.GetCentral was just called")
	}
	callInfo := struct {
		Ctx       context.Context
		CentralID string
	}{
		Ctx:       ctx,
		CentralID: centralID,
	}
	mock.lockGetCentral.Lock()
	mock.calls.GetCentral = append(mock.calls.GetCentral, callInfo)
	mock.lockGetCentral.Unlock()
	return mock.GetCentralFunc(ctx, centralID)
}

// GetCentralCalls gets all the calls that were made to GetCentral.
// Check the length with:
//
//	len(mockedPrivateAPI.GetCentralCalls())
func (mock *PrivateAPIMock) GetCentralCalls() []struct {
	Ctx       context.Context
	CentralID string
} {
	var calls []struct {
		Ctx       context.Context
		CentralID string
	}
	mock.lockGetCentral.RLock()
	calls = mock.calls.GetCentral
	mock.lockGetCentral.RUnlock()
	return calls
}

// GetCentrals calls GetCentralsFunc.
func (mock *PrivateAPIMock) GetCentrals(ctx context.Context, id string) (private.ManagedCentralList, *http.Response, error) {
	if mock.GetCentralsFunc == nil {
		panic("PrivateAPIMock.GetCentralsFunc: method is nil but PrivateAPI.GetCentrals was just called")
	}
	callInfo := struct {
		Ctx context.Context
		ID  string
	}{
		Ctx: ctx,
		ID:  id,
	}
	mock.lockGetCentrals.Lock()
	mock.calls.GetCentrals = append(mock.calls.GetCentrals, callInfo)
	mock.lockGetCentrals.Unlock()
	return mock.GetCentralsFunc(ctx, id)
}

// GetCentralsCalls gets all the calls that were made to GetCentrals.
// Check the length with:
//
//	len(mockedPrivateAPI.GetCentralsCalls())
func (mock *PrivateAPIMock) GetCentralsCalls() []struct {
	Ctx context.Context
	ID  string
} {
	var calls []struct {
		Ctx context.Context
		ID  string
	}
	mock.lockGetCentrals.RLock()
	calls = mock.calls.GetCentrals
	mock.lockGetCentrals.RUnlock()
	return calls
}

// UpdateAgentClusterStatus calls UpdateAgentClusterStatusFunc.
func (mock *PrivateAPIMock) UpdateAgentClusterStatus(ctx context.Context, id string, request private.DataPlaneClusterUpdateStatusRequest) (*http.Response, error) {
	if mock.UpdateAgentClusterStatusFunc == nil {
		panic("PrivateAPIMock.UpdateAgentClusterStatusFunc: method is nil but PrivateAPI.UpdateAgentClusterStatus was just called")
	}
	callInfo := struct {
		Ctx     context.Context
		ID      string
		Request private.DataPlaneClusterUpdateStatusRequest
	}{
		Ctx:     ctx,
		ID:      id,
		Request: request,
	}
	mock.lockUpdateAgentClusterStatus.Lock()
	mock.calls.UpdateAgentClusterStatus = append(mock.calls.UpdateAgentClusterStatus, callInfo)
	mock.lockUpdateAgentClusterStatus.Unlock()
	return mock.UpdateAgentClusterStatusFunc(ctx, id, request)
}

// UpdateAgentClusterStatusCalls gets all the calls that were made to UpdateAgentClusterStatus.
// Check the length with:
//
//	len(mockedPrivateAPI.UpdateAgentClusterStatusCalls())
func (mock *PrivateAPIMock) UpdateAgentClusterStatusCalls() []struct {
	Ctx     context.Context
	ID      string
	Request private.DataPlaneClusterUpdateStatusRequest
} {
	var calls []struct {
		Ctx     context.Context
		ID      string
		Request private.DataPlaneClusterUpdateStatusRequest
	}
	mock.lockUpdateAgentClusterStatus.RLock()
	calls = mock.calls.UpdateAgentClusterStatus
	mock.lockUpdateAgentClusterStatus.RUnlock()
	return calls
}

// UpdateCentralClusterStatus calls UpdateCentralClusterStatusFunc.
func (mock *PrivateAPIMock) UpdateCentralClusterStatus(ctx context.Context, id string, requestBody map[string]private.DataPlaneCentralStatus) (*http.Response, error) {
	if mock.UpdateCentralClusterStatusFunc == nil {
		panic("PrivateAPIMock.UpdateCentralClusterStatusFunc: method is nil but PrivateAPI.UpdateCentralClusterStatus was just called")
	}
	callInfo := struct {
		Ctx         context.Context
		ID          string
		RequestBody map[string]private.DataPlaneCentralStatus
	}{
		Ctx:         ctx,
		ID:          id,
		RequestBody: requestBody,
	}
	mock.lockUpdateCentralClusterStatus.Lock()
	mock.calls.UpdateCentralClusterStatus = append(mock.calls.UpdateCentralClusterStatus, callInfo)
	mock.lockUpdateCentralClusterStatus.Unlock()
	return mock.UpdateCentralClusterStatusFunc(ctx, id, requestBody)
}

// UpdateCentralClusterStatusCalls gets all the calls that were made to UpdateCentralClusterStatus.
// Check the length with:
//
//	len(mockedPrivateAPI.UpdateCentralClusterStatusCalls())
func (mock *PrivateAPIMock) UpdateCentralClusterStatusCalls() []struct {
	Ctx         context.Context
	ID          string
	RequestBody map[string]private.DataPlaneCentralStatus
} {
	var calls []struct {
		Ctx         context.Context
		ID          string
		RequestBody map[string]private.DataPlaneCentralStatus
	}
	mock.lockUpdateCentralClusterStatus.RLock()
	calls = mock.calls.UpdateCentralClusterStatus
	mock.lockUpdateCentralClusterStatus.RUnlock()
	return calls
}

// Ensure, that AdminAPIMock does implement fleetmanager.AdminAPI.
// If this is not the case, regenerate this file with moq.
var _ fleetmanager.AdminAPI = &AdminAPIMock{}

// AdminAPIMock is a mock implementation of fleetmanager.AdminAPI.
//
//	func TestSomethingThatUsesAdminAPI(t *testing.T) {
//
//		// make and configure a mocked fleetmanager.AdminAPI
//		mockedAdminAPI := &AdminAPIMock{
//			AssignCentralClusterFunc: func(ctx context.Context, id string, centralAssignClusterRequest admin.CentralAssignClusterRequest) (*http.Response, error) {
//				panic("mock out the AssignCentralCluster method")
//			},
//			CentralRotateSecretsFunc: func(ctx context.Context, id string, centralRotateSecretsRequest admin.CentralRotateSecretsRequest) (*http.Response, error) {
//				panic("mock out the CentralRotateSecrets method")
//			},
//			CreateCentralFunc: func(ctx context.Context, async bool, centralRequestPayload admin.CentralRequestPayload) (admin.CentralRequest, *http.Response, error) {
//				panic("mock out the CreateCentral method")
//			},
//			DeleteDbCentralByIdFunc: func(ctx context.Context, id string) (*http.Response, error) {
//				panic("mock out the DeleteDbCentralById method")
//			},
//			GetCentralsFunc: func(ctx context.Context, localVarOptionals *admin.GetCentralsOpts) (admin.CentralList, *http.Response, error) {
//				panic("mock out the GetCentrals method")
//			},
//			RestoreCentralFunc: func(ctx context.Context, id string) (*http.Response, error) {
//				panic("mock out the RestoreCentral method")
//			},
//			UpdateCentralNameByIdFunc: func(ctx context.Context, id string, centralUpdateNameRequest admin.CentralUpdateNameRequest) (admin.Central, *http.Response, error) {
//				panic("mock out the UpdateCentralNameById method")
//			},
//		}
//
//		// use mockedAdminAPI in code that requires fleetmanager.AdminAPI
//		// and then make assertions.
//
//	}
type AdminAPIMock struct {
	// AssignCentralClusterFunc mocks the AssignCentralCluster method.
	AssignCentralClusterFunc func(ctx context.Context, id string, centralAssignClusterRequest admin.CentralAssignClusterRequest) (*http.Response, error)

	// CentralRotateSecretsFunc mocks the CentralRotateSecrets method.
	CentralRotateSecretsFunc func(ctx context.Context, id string, centralRotateSecretsRequest admin.CentralRotateSecretsRequest) (*http.Response, error)

	// CreateCentralFunc mocks the CreateCentral method.
	CreateCentralFunc func(ctx context.Context, async bool, centralRequestPayload admin.CentralRequestPayload) (admin.CentralRequest, *http.Response, error)

	// DeleteDbCentralByIdFunc mocks the DeleteDbCentralById method.
	DeleteDbCentralByIdFunc func(ctx context.Context, id string) (*http.Response, error)

	// GetCentralsFunc mocks the GetCentrals method.
	GetCentralsFunc func(ctx context.Context, localVarOptionals *admin.GetCentralsOpts) (admin.CentralList, *http.Response, error)

	// RestoreCentralFunc mocks the RestoreCentral method.
	RestoreCentralFunc func(ctx context.Context, id string) (*http.Response, error)

	// UpdateCentralNameByIdFunc mocks the UpdateCentralNameById method.
	UpdateCentralNameByIdFunc func(ctx context.Context, id string, centralUpdateNameRequest admin.CentralUpdateNameRequest) (admin.Central, *http.Response, error)

	// calls tracks calls to the methods.
	calls struct {
		// AssignCentralCluster holds details about calls to the AssignCentralCluster method.
		AssignCentralCluster []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// CentralAssignClusterRequest is the centralAssignClusterRequest argument value.
			CentralAssignClusterRequest admin.CentralAssignClusterRequest
		}
		// CentralRotateSecrets holds details about calls to the CentralRotateSecrets method.
		CentralRotateSecrets []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// CentralRotateSecretsRequest is the centralRotateSecretsRequest argument value.
			CentralRotateSecretsRequest admin.CentralRotateSecretsRequest
		}
		// CreateCentral holds details about calls to the CreateCentral method.
		CreateCentral []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Async is the async argument value.
			Async bool
			// CentralRequestPayload is the centralRequestPayload argument value.
			CentralRequestPayload admin.CentralRequestPayload
		}
		// DeleteDbCentralById holds details about calls to the DeleteDbCentralById method.
		DeleteDbCentralById []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
		}
		// GetCentrals holds details about calls to the GetCentrals method.
		GetCentrals []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// LocalVarOptionals is the localVarOptionals argument value.
			LocalVarOptionals *admin.GetCentralsOpts
		}
		// RestoreCentral holds details about calls to the RestoreCentral method.
		RestoreCentral []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
		}
		// UpdateCentralNameById holds details about calls to the UpdateCentralNameById method.
		UpdateCentralNameById []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// CentralUpdateNameRequest is the centralUpdateNameRequest argument value.
			CentralUpdateNameRequest admin.CentralUpdateNameRequest
		}
	}
	lockAssignCentralCluster  sync.RWMutex
	lockCentralRotateSecrets  sync.RWMutex
	lockCreateCentral         sync.RWMutex
	lockDeleteDbCentralById   sync.RWMutex
	lockGetCentrals           sync.RWMutex
	lockRestoreCentral        sync.RWMutex
	lockUpdateCentralNameById sync.RWMutex
}

// AssignCentralCluster calls AssignCentralClusterFunc.
func (mock *AdminAPIMock) AssignCentralCluster(ctx context.Context, id string, centralAssignClusterRequest admin.CentralAssignClusterRequest) (*http.Response, error) {
	if mock.AssignCentralClusterFunc == nil {
		panic("AdminAPIMock.AssignCentralClusterFunc: method is nil but AdminAPI.AssignCentralCluster was just called")
	}
	callInfo := struct {
		Ctx                         context.Context
		ID                          string
		CentralAssignClusterRequest admin.CentralAssignClusterRequest
	}{
		Ctx:                         ctx,
		ID:                          id,
		CentralAssignClusterRequest: centralAssignClusterRequest,
	}
	mock.lockAssignCentralCluster.Lock()
	mock.calls.AssignCentralCluster = append(mock.calls.AssignCentralCluster, callInfo)
	mock.lockAssignCentralCluster.Unlock()
	return mock.AssignCentralClusterFunc(ctx, id, centralAssignClusterRequest)
}

// AssignCentralClusterCalls gets all the calls that were made to AssignCentralCluster.
// Check the length with:
//
//	len(mockedAdminAPI.AssignCentralClusterCalls())
func (mock *AdminAPIMock) AssignCentralClusterCalls() []struct {
	Ctx                         context.Context
	ID                          string
	CentralAssignClusterRequest admin.CentralAssignClusterRequest
} {
	var calls []struct {
		Ctx                         context.Context
		ID                          string
		CentralAssignClusterRequest admin.CentralAssignClusterRequest
	}
	mock.lockAssignCentralCluster.RLock()
	calls = mock.calls.AssignCentralCluster
	mock.lockAssignCentralCluster.RUnlock()
	return calls
}

// CentralRotateSecrets calls CentralRotateSecretsFunc.
func (mock *AdminAPIMock) CentralRotateSecrets(ctx context.Context, id string, centralRotateSecretsRequest admin.CentralRotateSecretsRequest) (*http.Response, error) {
	if mock.CentralRotateSecretsFunc == nil {
		panic("AdminAPIMock.CentralRotateSecretsFunc: method is nil but AdminAPI.CentralRotateSecrets was just called")
	}
	callInfo := struct {
		Ctx                         context.Context
		ID                          string
		CentralRotateSecretsRequest admin.CentralRotateSecretsRequest
	}{
		Ctx:                         ctx,
		ID:                          id,
		CentralRotateSecretsRequest: centralRotateSecretsRequest,
	}
	mock.lockCentralRotateSecrets.Lock()
	mock.calls.CentralRotateSecrets = append(mock.calls.CentralRotateSecrets, callInfo)
	mock.lockCentralRotateSecrets.Unlock()
	return mock.CentralRotateSecretsFunc(ctx, id, centralRotateSecretsRequest)
}

// CentralRotateSecretsCalls gets all the calls that were made to CentralRotateSecrets.
// Check the length with:
//
//	len(mockedAdminAPI.CentralRotateSecretsCalls())
func (mock *AdminAPIMock) CentralRotateSecretsCalls() []struct {
	Ctx                         context.Context
	ID                          string
	CentralRotateSecretsRequest admin.CentralRotateSecretsRequest
} {
	var calls []struct {
		Ctx                         context.Context
		ID                          string
		CentralRotateSecretsRequest admin.CentralRotateSecretsRequest
	}
	mock.lockCentralRotateSecrets.RLock()
	calls = mock.calls.CentralRotateSecrets
	mock.lockCentralRotateSecrets.RUnlock()
	return calls
}

// CreateCentral calls CreateCentralFunc.
func (mock *AdminAPIMock) CreateCentral(ctx context.Context, async bool, centralRequestPayload admin.CentralRequestPayload) (admin.CentralRequest, *http.Response, error) {
	if mock.CreateCentralFunc == nil {
		panic("AdminAPIMock.CreateCentralFunc: method is nil but AdminAPI.CreateCentral was just called")
	}
	callInfo := struct {
		Ctx                   context.Context
		Async                 bool
		CentralRequestPayload admin.CentralRequestPayload
	}{
		Ctx:                   ctx,
		Async:                 async,
		CentralRequestPayload: centralRequestPayload,
	}
	mock.lockCreateCentral.Lock()
	mock.calls.CreateCentral = append(mock.calls.CreateCentral, callInfo)
	mock.lockCreateCentral.Unlock()
	return mock.CreateCentralFunc(ctx, async, centralRequestPayload)
}

// CreateCentralCalls gets all the calls that were made to CreateCentral.
// Check the length with:
//
//	len(mockedAdminAPI.CreateCentralCalls())
func (mock *AdminAPIMock) CreateCentralCalls() []struct {
	Ctx                   context.Context
	Async                 bool
	CentralRequestPayload admin.CentralRequestPayload
} {
	var calls []struct {
		Ctx                   context.Context
		Async                 bool
		CentralRequestPayload admin.CentralRequestPayload
	}
	mock.lockCreateCentral.RLock()
	calls = mock.calls.CreateCentral
	mock.lockCreateCentral.RUnlock()
	return calls
}

// DeleteDbCentralById calls DeleteDbCentralByIdFunc.
func (mock *AdminAPIMock) DeleteDbCentralById(ctx context.Context, id string) (*http.Response, error) {
	if mock.DeleteDbCentralByIdFunc == nil {
		panic("AdminAPIMock.DeleteDbCentralByIdFunc: method is nil but AdminAPI.DeleteDbCentralById was just called")
	}
	callInfo := struct {
		Ctx context.Context
		ID  string
	}{
		Ctx: ctx,
		ID:  id,
	}
	mock.lockDeleteDbCentralById.Lock()
	mock.calls.DeleteDbCentralById = append(mock.calls.DeleteDbCentralById, callInfo)
	mock.lockDeleteDbCentralById.Unlock()
	return mock.DeleteDbCentralByIdFunc(ctx, id)
}

// DeleteDbCentralByIdCalls gets all the calls that were made to DeleteDbCentralById.
// Check the length with:
//
//	len(mockedAdminAPI.DeleteDbCentralByIdCalls())
func (mock *AdminAPIMock) DeleteDbCentralByIdCalls() []struct {
	Ctx context.Context
	ID  string
} {
	var calls []struct {
		Ctx context.Context
		ID  string
	}
	mock.lockDeleteDbCentralById.RLock()
	calls = mock.calls.DeleteDbCentralById
	mock.lockDeleteDbCentralById.RUnlock()
	return calls
}

// GetCentrals calls GetCentralsFunc.
func (mock *AdminAPIMock) GetCentrals(ctx context.Context, localVarOptionals *admin.GetCentralsOpts) (admin.CentralList, *http.Response, error) {
	if mock.GetCentralsFunc == nil {
		panic("AdminAPIMock.GetCentralsFunc: method is nil but AdminAPI.GetCentrals was just called")
	}
	callInfo := struct {
		Ctx               context.Context
		LocalVarOptionals *admin.GetCentralsOpts
	}{
		Ctx:               ctx,
		LocalVarOptionals: localVarOptionals,
	}
	mock.lockGetCentrals.Lock()
	mock.calls.GetCentrals = append(mock.calls.GetCentrals, callInfo)
	mock.lockGetCentrals.Unlock()
	return mock.GetCentralsFunc(ctx, localVarOptionals)
}

// GetCentralsCalls gets all the calls that were made to GetCentrals.
// Check the length with:
//
//	len(mockedAdminAPI.GetCentralsCalls())
func (mock *AdminAPIMock) GetCentralsCalls() []struct {
	Ctx               context.Context
	LocalVarOptionals *admin.GetCentralsOpts
} {
	var calls []struct {
		Ctx               context.Context
		LocalVarOptionals *admin.GetCentralsOpts
	}
	mock.lockGetCentrals.RLock()
	calls = mock.calls.GetCentrals
	mock.lockGetCentrals.RUnlock()
	return calls
}

// RestoreCentral calls RestoreCentralFunc.
func (mock *AdminAPIMock) RestoreCentral(ctx context.Context, id string) (*http.Response, error) {
	if mock.RestoreCentralFunc == nil {
		panic("AdminAPIMock.RestoreCentralFunc: method is nil but AdminAPI.RestoreCentral was just called")
	}
	callInfo := struct {
		Ctx context.Context
		ID  string
	}{
		Ctx: ctx,
		ID:  id,
	}
	mock.lockRestoreCentral.Lock()
	mock.calls.RestoreCentral = append(mock.calls.RestoreCentral, callInfo)
	mock.lockRestoreCentral.Unlock()
	return mock.RestoreCentralFunc(ctx, id)
}

// RestoreCentralCalls gets all the calls that were made to RestoreCentral.
// Check the length with:
//
//	len(mockedAdminAPI.RestoreCentralCalls())
func (mock *AdminAPIMock) RestoreCentralCalls() []struct {
	Ctx context.Context
	ID  string
} {
	var calls []struct {
		Ctx context.Context
		ID  string
	}
	mock.lockRestoreCentral.RLock()
	calls = mock.calls.RestoreCentral
	mock.lockRestoreCentral.RUnlock()
	return calls
}

// UpdateCentralNameById calls UpdateCentralNameByIdFunc.
func (mock *AdminAPIMock) UpdateCentralNameById(ctx context.Context, id string, centralUpdateNameRequest admin.CentralUpdateNameRequest) (admin.Central, *http.Response, error) {
	if mock.UpdateCentralNameByIdFunc == nil {
		panic("AdminAPIMock.UpdateCentralNameByIdFunc: method is nil but AdminAPI.UpdateCentralNameById was just called")
	}
	callInfo := struct {
		Ctx                      context.Context
		ID                       string
		CentralUpdateNameRequest admin.CentralUpdateNameRequest
	}{
		Ctx:                      ctx,
		ID:                       id,
		CentralUpdateNameRequest: centralUpdateNameRequest,
	}
	mock.lockUpdateCentralNameById.Lock()
	mock.calls.UpdateCentralNameById = append(mock.calls.UpdateCentralNameById, callInfo)
	mock.lockUpdateCentralNameById.Unlock()
	return mock.UpdateCentralNameByIdFunc(ctx, id, centralUpdateNameRequest)
}

// UpdateCentralNameByIdCalls gets all the calls that were made to UpdateCentralNameById.
// Check the length with:
//
//	len(mockedAdminAPI.UpdateCentralNameByIdCalls())
func (mock *AdminAPIMock) UpdateCentralNameByIdCalls() []struct {
	Ctx                      context.Context
	ID                       string
	CentralUpdateNameRequest admin.CentralUpdateNameRequest
} {
	var calls []struct {
		Ctx                      context.Context
		ID                       string
		CentralUpdateNameRequest admin.CentralUpdateNameRequest
	}
	mock.lockUpdateCentralNameById.RLock()
	calls = mock.calls.UpdateCentralNameById
	mock.lockUpdateCentralNameById.RUnlock()
	return calls
}
