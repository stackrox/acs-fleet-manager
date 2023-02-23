// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package sso

import (
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"sync"
)

// Ensure, that IAMServiceMock does implement IAMService.
// If this is not the case, regenerate this file with moq.
var _ IAMService = &IAMServiceMock{}

// IAMServiceMock is a mock implementation of IAMService.
//
//	func TestSomethingThatUsesIAMService(t *testing.T) {
//
//		// make and configure a mocked IAMService
//		mockedIAMService := &IAMServiceMock{
//			DeRegisterAcsFleetshardOperatorServiceAccountFunc: func(agentClusterID string) *errors.ServiceError {
//				panic("mock out the DeRegisterAcsFleetshardOperatorServiceAccount method")
//			},
//			RegisterAcsFleetshardOperatorServiceAccountFunc: func(agentClusterID string) (*api.ServiceAccount, *errors.ServiceError) {
//				panic("mock out the RegisterAcsFleetshardOperatorServiceAccount method")
//			},
//		}
//
//		// use mockedIAMService in code that requires IAMService
//		// and then make assertions.
//
//	}
type IAMServiceMock struct {
	// DeRegisterAcsFleetshardOperatorServiceAccountFunc mocks the DeRegisterAcsFleetshardOperatorServiceAccount method.
	DeRegisterAcsFleetshardOperatorServiceAccountFunc func(agentClusterID string) *errors.ServiceError

	// RegisterAcsFleetshardOperatorServiceAccountFunc mocks the RegisterAcsFleetshardOperatorServiceAccount method.
	RegisterAcsFleetshardOperatorServiceAccountFunc func(agentClusterID string) (*api.ServiceAccount, *errors.ServiceError)

	// calls tracks calls to the methods.
	calls struct {
		// DeRegisterAcsFleetshardOperatorServiceAccount holds details about calls to the DeRegisterAcsFleetshardOperatorServiceAccount method.
		DeRegisterAcsFleetshardOperatorServiceAccount []struct {
			// AgentClusterID is the agentClusterID argument value.
			AgentClusterID string
		}
		// RegisterAcsFleetshardOperatorServiceAccount holds details about calls to the RegisterAcsFleetshardOperatorServiceAccount method.
		RegisterAcsFleetshardOperatorServiceAccount []struct {
			// AgentClusterID is the agentClusterID argument value.
			AgentClusterID string
		}
	}
	lockDeRegisterAcsFleetshardOperatorServiceAccount sync.RWMutex
	lockRegisterAcsFleetshardOperatorServiceAccount   sync.RWMutex
}

// DeRegisterAcsFleetshardOperatorServiceAccount calls DeRegisterAcsFleetshardOperatorServiceAccountFunc.
func (mock *IAMServiceMock) DeRegisterAcsFleetshardOperatorServiceAccount(agentClusterID string) *errors.ServiceError {
	if mock.DeRegisterAcsFleetshardOperatorServiceAccountFunc == nil {
		panic("IAMServiceMock.DeRegisterAcsFleetshardOperatorServiceAccountFunc: method is nil but IAMService.DeRegisterAcsFleetshardOperatorServiceAccount was just called")
	}
	callInfo := struct {
		AgentClusterID string
	}{
		AgentClusterID: agentClusterID,
	}
	mock.lockDeRegisterAcsFleetshardOperatorServiceAccount.Lock()
	mock.calls.DeRegisterAcsFleetshardOperatorServiceAccount = append(mock.calls.DeRegisterAcsFleetshardOperatorServiceAccount, callInfo)
	mock.lockDeRegisterAcsFleetshardOperatorServiceAccount.Unlock()
	return mock.DeRegisterAcsFleetshardOperatorServiceAccountFunc(agentClusterID)
}

// DeRegisterAcsFleetshardOperatorServiceAccountCalls gets all the calls that were made to DeRegisterAcsFleetshardOperatorServiceAccount.
// Check the length with:
//
//	len(mockedIAMService.DeRegisterAcsFleetshardOperatorServiceAccountCalls())
func (mock *IAMServiceMock) DeRegisterAcsFleetshardOperatorServiceAccountCalls() []struct {
	AgentClusterID string
} {
	var calls []struct {
		AgentClusterID string
	}
	mock.lockDeRegisterAcsFleetshardOperatorServiceAccount.RLock()
	calls = mock.calls.DeRegisterAcsFleetshardOperatorServiceAccount
	mock.lockDeRegisterAcsFleetshardOperatorServiceAccount.RUnlock()
	return calls
}

// RegisterAcsFleetshardOperatorServiceAccount calls RegisterAcsFleetshardOperatorServiceAccountFunc.
func (mock *IAMServiceMock) RegisterAcsFleetshardOperatorServiceAccount(agentClusterID string) (*api.ServiceAccount, *errors.ServiceError) {
	if mock.RegisterAcsFleetshardOperatorServiceAccountFunc == nil {
		panic("IAMServiceMock.RegisterAcsFleetshardOperatorServiceAccountFunc: method is nil but IAMService.RegisterAcsFleetshardOperatorServiceAccount was just called")
	}
	callInfo := struct {
		AgentClusterID string
	}{
		AgentClusterID: agentClusterID,
	}
	mock.lockRegisterAcsFleetshardOperatorServiceAccount.Lock()
	mock.calls.RegisterAcsFleetshardOperatorServiceAccount = append(mock.calls.RegisterAcsFleetshardOperatorServiceAccount, callInfo)
	mock.lockRegisterAcsFleetshardOperatorServiceAccount.Unlock()
	return mock.RegisterAcsFleetshardOperatorServiceAccountFunc(agentClusterID)
}

// RegisterAcsFleetshardOperatorServiceAccountCalls gets all the calls that were made to RegisterAcsFleetshardOperatorServiceAccount.
// Check the length with:
//
//	len(mockedIAMService.RegisterAcsFleetshardOperatorServiceAccountCalls())
func (mock *IAMServiceMock) RegisterAcsFleetshardOperatorServiceAccountCalls() []struct {
	AgentClusterID string
} {
	var calls []struct {
		AgentClusterID string
	}
	mock.lockRegisterAcsFleetshardOperatorServiceAccount.RLock()
	calls = mock.calls.RegisterAcsFleetshardOperatorServiceAccount
	mock.lockRegisterAcsFleetshardOperatorServiceAccount.RUnlock()
	return calls
}
