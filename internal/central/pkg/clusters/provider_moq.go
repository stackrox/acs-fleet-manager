// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package clusters

import (
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/clusters/types"
	"sync"
)

// Ensure, that ProviderMock does implement Provider.
// If this is not the case, regenerate this file with moq.
var _ Provider = &ProviderMock{}

// ProviderMock is a mock implementation of Provider.
//
//	func TestSomethingThatUsesProvider(t *testing.T) {
//
//		// make and configure a mocked Provider
//		mockedProvider := &ProviderMock{
//			CheckClusterStatusFunc: func(spec *types.ClusterSpec) (*types.ClusterSpec, error) {
//				panic("mock out the CheckClusterStatus method")
//			},
//			CreateFunc: func(request *types.ClusterRequest) (*types.ClusterSpec, error) {
//				panic("mock out the Create method")
//			},
//			DeleteFunc: func(spec *types.ClusterSpec) (bool, error) {
//				panic("mock out the Delete method")
//			},
//			GetCloudProviderRegionsFunc: func(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
//				panic("mock out the GetCloudProviderRegions method")
//			},
//			GetCloudProvidersFunc: func() (*types.CloudProviderInfoList, error) {
//				panic("mock out the GetCloudProviders method")
//			},
//			GetClusterDNSFunc: func(clusterSpec *types.ClusterSpec) (string, error) {
//				panic("mock out the GetClusterDNS method")
//			},
//		}
//
//		// use mockedProvider in code that requires Provider
//		// and then make assertions.
//
//	}
type ProviderMock struct {
	// CheckClusterStatusFunc mocks the CheckClusterStatus method.
	CheckClusterStatusFunc func(spec *types.ClusterSpec) (*types.ClusterSpec, error)

	// CreateFunc mocks the Create method.
	CreateFunc func(request *types.ClusterRequest) (*types.ClusterSpec, error)

	// DeleteFunc mocks the Delete method.
	DeleteFunc func(spec *types.ClusterSpec) (bool, error)

	// GetCloudProviderRegionsFunc mocks the GetCloudProviderRegions method.
	GetCloudProviderRegionsFunc func(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error)

	// GetCloudProvidersFunc mocks the GetCloudProviders method.
	GetCloudProvidersFunc func() (*types.CloudProviderInfoList, error)

	// GetClusterDNSFunc mocks the GetClusterDNS method.
	GetClusterDNSFunc func(clusterSpec *types.ClusterSpec) (string, error)

	// calls tracks calls to the methods.
	calls struct {
		// CheckClusterStatus holds details about calls to the CheckClusterStatus method.
		CheckClusterStatus []struct {
			// Spec is the spec argument value.
			Spec *types.ClusterSpec
		}
		// Create holds details about calls to the Create method.
		Create []struct {
			// Request is the request argument value.
			Request *types.ClusterRequest
		}
		// Delete holds details about calls to the Delete method.
		Delete []struct {
			// Spec is the spec argument value.
			Spec *types.ClusterSpec
		}
		// GetCloudProviderRegions holds details about calls to the GetCloudProviderRegions method.
		GetCloudProviderRegions []struct {
			// ProviderInf is the providerInf argument value.
			ProviderInf types.CloudProviderInfo
		}
		// GetCloudProviders holds details about calls to the GetCloudProviders method.
		GetCloudProviders []struct {
		}
		// GetClusterDNS holds details about calls to the GetClusterDNS method.
		GetClusterDNS []struct {
			// ClusterSpec is the clusterSpec argument value.
			ClusterSpec *types.ClusterSpec
		}
	}
	lockCheckClusterStatus      sync.RWMutex
	lockCreate                  sync.RWMutex
	lockDelete                  sync.RWMutex
	lockGetCloudProviderRegions sync.RWMutex
	lockGetCloudProviders       sync.RWMutex
	lockGetClusterDNS           sync.RWMutex
}

// CheckClusterStatus calls CheckClusterStatusFunc.
func (mock *ProviderMock) CheckClusterStatus(spec *types.ClusterSpec) (*types.ClusterSpec, error) {
	if mock.CheckClusterStatusFunc == nil {
		panic("ProviderMock.CheckClusterStatusFunc: method is nil but Provider.CheckClusterStatus was just called")
	}
	callInfo := struct {
		Spec *types.ClusterSpec
	}{
		Spec: spec,
	}
	mock.lockCheckClusterStatus.Lock()
	mock.calls.CheckClusterStatus = append(mock.calls.CheckClusterStatus, callInfo)
	mock.lockCheckClusterStatus.Unlock()
	return mock.CheckClusterStatusFunc(spec)
}

// CheckClusterStatusCalls gets all the calls that were made to CheckClusterStatus.
// Check the length with:
//
//	len(mockedProvider.CheckClusterStatusCalls())
func (mock *ProviderMock) CheckClusterStatusCalls() []struct {
	Spec *types.ClusterSpec
} {
	var calls []struct {
		Spec *types.ClusterSpec
	}
	mock.lockCheckClusterStatus.RLock()
	calls = mock.calls.CheckClusterStatus
	mock.lockCheckClusterStatus.RUnlock()
	return calls
}

// Create calls CreateFunc.
func (mock *ProviderMock) Create(request *types.ClusterRequest) (*types.ClusterSpec, error) {
	if mock.CreateFunc == nil {
		panic("ProviderMock.CreateFunc: method is nil but Provider.Create was just called")
	}
	callInfo := struct {
		Request *types.ClusterRequest
	}{
		Request: request,
	}
	mock.lockCreate.Lock()
	mock.calls.Create = append(mock.calls.Create, callInfo)
	mock.lockCreate.Unlock()
	return mock.CreateFunc(request)
}

// CreateCalls gets all the calls that were made to Create.
// Check the length with:
//
//	len(mockedProvider.CreateCalls())
func (mock *ProviderMock) CreateCalls() []struct {
	Request *types.ClusterRequest
} {
	var calls []struct {
		Request *types.ClusterRequest
	}
	mock.lockCreate.RLock()
	calls = mock.calls.Create
	mock.lockCreate.RUnlock()
	return calls
}

// Delete calls DeleteFunc.
func (mock *ProviderMock) Delete(spec *types.ClusterSpec) (bool, error) {
	if mock.DeleteFunc == nil {
		panic("ProviderMock.DeleteFunc: method is nil but Provider.Delete was just called")
	}
	callInfo := struct {
		Spec *types.ClusterSpec
	}{
		Spec: spec,
	}
	mock.lockDelete.Lock()
	mock.calls.Delete = append(mock.calls.Delete, callInfo)
	mock.lockDelete.Unlock()
	return mock.DeleteFunc(spec)
}

// DeleteCalls gets all the calls that were made to Delete.
// Check the length with:
//
//	len(mockedProvider.DeleteCalls())
func (mock *ProviderMock) DeleteCalls() []struct {
	Spec *types.ClusterSpec
} {
	var calls []struct {
		Spec *types.ClusterSpec
	}
	mock.lockDelete.RLock()
	calls = mock.calls.Delete
	mock.lockDelete.RUnlock()
	return calls
}

// GetCloudProviderRegions calls GetCloudProviderRegionsFunc.
func (mock *ProviderMock) GetCloudProviderRegions(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
	if mock.GetCloudProviderRegionsFunc == nil {
		panic("ProviderMock.GetCloudProviderRegionsFunc: method is nil but Provider.GetCloudProviderRegions was just called")
	}
	callInfo := struct {
		ProviderInf types.CloudProviderInfo
	}{
		ProviderInf: providerInf,
	}
	mock.lockGetCloudProviderRegions.Lock()
	mock.calls.GetCloudProviderRegions = append(mock.calls.GetCloudProviderRegions, callInfo)
	mock.lockGetCloudProviderRegions.Unlock()
	return mock.GetCloudProviderRegionsFunc(providerInf)
}

// GetCloudProviderRegionsCalls gets all the calls that were made to GetCloudProviderRegions.
// Check the length with:
//
//	len(mockedProvider.GetCloudProviderRegionsCalls())
func (mock *ProviderMock) GetCloudProviderRegionsCalls() []struct {
	ProviderInf types.CloudProviderInfo
} {
	var calls []struct {
		ProviderInf types.CloudProviderInfo
	}
	mock.lockGetCloudProviderRegions.RLock()
	calls = mock.calls.GetCloudProviderRegions
	mock.lockGetCloudProviderRegions.RUnlock()
	return calls
}

// GetCloudProviders calls GetCloudProvidersFunc.
func (mock *ProviderMock) GetCloudProviders() (*types.CloudProviderInfoList, error) {
	if mock.GetCloudProvidersFunc == nil {
		panic("ProviderMock.GetCloudProvidersFunc: method is nil but Provider.GetCloudProviders was just called")
	}
	callInfo := struct {
	}{}
	mock.lockGetCloudProviders.Lock()
	mock.calls.GetCloudProviders = append(mock.calls.GetCloudProviders, callInfo)
	mock.lockGetCloudProviders.Unlock()
	return mock.GetCloudProvidersFunc()
}

// GetCloudProvidersCalls gets all the calls that were made to GetCloudProviders.
// Check the length with:
//
//	len(mockedProvider.GetCloudProvidersCalls())
func (mock *ProviderMock) GetCloudProvidersCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGetCloudProviders.RLock()
	calls = mock.calls.GetCloudProviders
	mock.lockGetCloudProviders.RUnlock()
	return calls
}

// GetClusterDNS calls GetClusterDNSFunc.
func (mock *ProviderMock) GetClusterDNS(clusterSpec *types.ClusterSpec) (string, error) {
	if mock.GetClusterDNSFunc == nil {
		panic("ProviderMock.GetClusterDNSFunc: method is nil but Provider.GetClusterDNS was just called")
	}
	callInfo := struct {
		ClusterSpec *types.ClusterSpec
	}{
		ClusterSpec: clusterSpec,
	}
	mock.lockGetClusterDNS.Lock()
	mock.calls.GetClusterDNS = append(mock.calls.GetClusterDNS, callInfo)
	mock.lockGetClusterDNS.Unlock()
	return mock.GetClusterDNSFunc(clusterSpec)
}

// GetClusterDNSCalls gets all the calls that were made to GetClusterDNS.
// Check the length with:
//
//	len(mockedProvider.GetClusterDNSCalls())
func (mock *ProviderMock) GetClusterDNSCalls() []struct {
	ClusterSpec *types.ClusterSpec
} {
	var calls []struct {
		ClusterSpec *types.ClusterSpec
	}
	mock.lockGetClusterDNS.RLock()
	calls = mock.calls.GetClusterDNS
	mock.lockGetClusterDNS.RUnlock()
	return calls
}
