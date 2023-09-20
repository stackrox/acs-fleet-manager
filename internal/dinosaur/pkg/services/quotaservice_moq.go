// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package services

import (
	"context"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	serviceError "github.com/stackrox/acs-fleet-manager/pkg/errors"
	"sync"
)

// Ensure, that QuotaServiceMock does implement QuotaService.
// If this is not the case, regenerate this file with moq.
var _ QuotaService = &QuotaServiceMock{}

// QuotaServiceMock is a mock implementation of QuotaService.
//
//	func TestSomethingThatUsesQuotaService(t *testing.T) {
//
//		// make and configure a mocked QuotaService
//		mockedQuotaService := &QuotaServiceMock{
//			CheckIfQuotaIsDefinedForInstanceTypeFunc: func(dinosaur *dbapi.CentralRequest, instanceType types.DinosaurInstanceType) (bool, *serviceError.ServiceError) {
//				panic("mock out the CheckIfQuotaIsDefinedForInstanceType method")
//			},
//			DeleteQuotaFunc: func(subscriptionID string) *serviceError.ServiceError {
//				panic("mock out the DeleteQuota method")
//			},
//			ReserveQuotaFunc: func(dinosaur *dbapi.CentralRequest, instanceType types.DinosaurInstanceType) (string, *serviceError.ServiceError) {
//				panic("mock out the ReserveQuota method")
//			},
//		}
//
//		// use mockedQuotaService in code that requires QuotaService
//		// and then make assertions.
//
//	}
type QuotaServiceMock struct {
	// CheckIfQuotaIsDefinedForInstanceTypeFunc mocks the CheckIfQuotaIsDefinedForInstanceType method.
	CheckIfQuotaIsDefinedForInstanceTypeFunc func(dinosaur *dbapi.CentralRequest, instanceType types.DinosaurInstanceType) (bool, *serviceError.ServiceError)

	// DeleteQuotaFunc mocks the DeleteQuota method.
	DeleteQuotaFunc func(subscriptionID string) *serviceError.ServiceError

	// ReserveQuotaFunc mocks the ReserveQuota method.
	ReserveQuotaFunc func(dinosaur *dbapi.CentralRequest, instanceType types.DinosaurInstanceType) (string, *serviceError.ServiceError)

	// calls tracks calls to the methods.
	calls struct {
		// CheckIfQuotaIsDefinedForInstanceType holds details about calls to the CheckIfQuotaIsDefinedForInstanceType method.
		CheckIfQuotaIsDefinedForInstanceType []struct {
			// Dinosaur is the dinosaur argument value.
			Dinosaur *dbapi.CentralRequest
			// InstanceType is the instanceType argument value.
			InstanceType types.DinosaurInstanceType
		}
		// DeleteQuota holds details about calls to the DeleteQuota method.
		DeleteQuota []struct {
			// SubscriptionID is the subscriptionID argument value.
			SubscriptionID string
		}
		// ReserveQuota holds details about calls to the ReserveQuota method.
		ReserveQuota []struct {
			// Dinosaur is the dinosaur argument value.
			Dinosaur *dbapi.CentralRequest
			// InstanceType is the instanceType argument value.
			InstanceType types.DinosaurInstanceType
		}
	}
	lockCheckIfQuotaIsDefinedForInstanceType sync.RWMutex
	lockDeleteQuota                          sync.RWMutex
	lockReserveQuota                         sync.RWMutex
}

// CheckIfQuotaIsDefinedForInstanceType calls CheckIfQuotaIsDefinedForInstanceTypeFunc.
func (mock *QuotaServiceMock) CheckIfQuotaIsDefinedForInstanceType(dinosaur *dbapi.CentralRequest, instanceType types.DinosaurInstanceType) (bool, *serviceError.ServiceError) {
	if mock.CheckIfQuotaIsDefinedForInstanceTypeFunc == nil {
		panic("QuotaServiceMock.CheckIfQuotaIsDefinedForInstanceTypeFunc: method is nil but QuotaService.CheckIfQuotaIsDefinedForInstanceType was just called")
	}
	callInfo := struct {
		Dinosaur     *dbapi.CentralRequest
		InstanceType types.DinosaurInstanceType
	}{
		Dinosaur:     dinosaur,
		InstanceType: instanceType,
	}
	mock.lockCheckIfQuotaIsDefinedForInstanceType.Lock()
	mock.calls.CheckIfQuotaIsDefinedForInstanceType = append(mock.calls.CheckIfQuotaIsDefinedForInstanceType, callInfo)
	mock.lockCheckIfQuotaIsDefinedForInstanceType.Unlock()
	return mock.CheckIfQuotaIsDefinedForInstanceTypeFunc(dinosaur, instanceType)
}

// CheckIfQuotaIsDefinedForInstanceTypeCalls gets all the calls that were made to CheckIfQuotaIsDefinedForInstanceType.
// Check the length with:
//
//	len(mockedQuotaService.CheckIfQuotaIsDefinedForInstanceTypeCalls())
func (mock *QuotaServiceMock) CheckIfQuotaIsDefinedForInstanceTypeCalls() []struct {
	Dinosaur     *dbapi.CentralRequest
	InstanceType types.DinosaurInstanceType
} {
	var calls []struct {
		Dinosaur     *dbapi.CentralRequest
		InstanceType types.DinosaurInstanceType
	}
	mock.lockCheckIfQuotaIsDefinedForInstanceType.RLock()
	calls = mock.calls.CheckIfQuotaIsDefinedForInstanceType
	mock.lockCheckIfQuotaIsDefinedForInstanceType.RUnlock()
	return calls
}

// DeleteQuota calls DeleteQuotaFunc.
func (mock *QuotaServiceMock) DeleteQuota(subscriptionID string) *serviceError.ServiceError {
	if mock.DeleteQuotaFunc == nil {
		panic("QuotaServiceMock.DeleteQuotaFunc: method is nil but QuotaService.DeleteQuota was just called")
	}
	callInfo := struct {
		SubscriptionID string
	}{
		SubscriptionID: subscriptionID,
	}
	mock.lockDeleteQuota.Lock()
	mock.calls.DeleteQuota = append(mock.calls.DeleteQuota, callInfo)
	mock.lockDeleteQuota.Unlock()
	return mock.DeleteQuotaFunc(subscriptionID)
}

// DeleteQuotaCalls gets all the calls that were made to DeleteQuota.
// Check the length with:
//
//	len(mockedQuotaService.DeleteQuotaCalls())
func (mock *QuotaServiceMock) DeleteQuotaCalls() []struct {
	SubscriptionID string
} {
	var calls []struct {
		SubscriptionID string
	}
	mock.lockDeleteQuota.RLock()
	calls = mock.calls.DeleteQuota
	mock.lockDeleteQuota.RUnlock()
	return calls
}

// ReserveQuota calls ReserveQuotaFunc.
func (mock *QuotaServiceMock) ReserveQuota(ctx context.Context, dinosaur *dbapi.CentralRequest, instanceType types.DinosaurInstanceType) (string, *serviceError.ServiceError) {
	if mock.ReserveQuotaFunc == nil {
		panic("QuotaServiceMock.ReserveQuotaFunc: method is nil but QuotaService.ReserveQuota was just called")
	}
	callInfo := struct {
		Dinosaur     *dbapi.CentralRequest
		InstanceType types.DinosaurInstanceType
	}{
		Dinosaur:     dinosaur,
		InstanceType: instanceType,
	}
	mock.lockReserveQuota.Lock()
	mock.calls.ReserveQuota = append(mock.calls.ReserveQuota, callInfo)
	mock.lockReserveQuota.Unlock()
	return mock.ReserveQuotaFunc(dinosaur, instanceType)
}

// ReserveQuotaCalls gets all the calls that were made to ReserveQuota.
// Check the length with:
//
//	len(mockedQuotaService.ReserveQuotaCalls())
func (mock *QuotaServiceMock) ReserveQuotaCalls() []struct {
	Dinosaur     *dbapi.CentralRequest
	InstanceType types.DinosaurInstanceType
} {
	var calls []struct {
		Dinosaur     *dbapi.CentralRequest
		InstanceType types.DinosaurInstanceType
	}
	mock.lockReserveQuota.RLock()
	calls = mock.calls.ReserveQuota
	mock.lockReserveQuota.RUnlock()
	return calls
}
