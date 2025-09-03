package centralmgrs

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCentralService is a mock implementation of services.CentralService
type MockCentralService struct {
	centrals         []*dbapi.CentralRequest
	updateCalled     bool
	cnameChangeErr   error
	cnameStatusErr   error
	cnameChangeInfo  *route53.ChangeResourceRecordSetsOutput
	cnameRecordStatus *route53.GetChangeOutput
}

func (m *MockCentralService) ListCentralsWithRoutesNotCreated() ([]*dbapi.CentralRequest, error) {
	return m.centrals, nil
}

func (m *MockCentralService) ChangeCentralCNAMErecords(central *dbapi.CentralRequest, action services.CentralRoutesAction) (*route53.ChangeResourceRecordSetsOutput, error) {
	return m.cnameChangeInfo, m.cnameChangeErr
}

func (m *MockCentralService) GetCNAMERecordStatus(central *dbapi.CentralRequest) (*route53.GetChangeOutput, error) {
	return m.cnameRecordStatus, m.cnameStatusErr
}

func (m *MockCentralService) UpdateIgnoreNils(central *dbapi.CentralRequest) error {
	m.updateCalled = true
	return nil
}

// Implement other required methods with empty implementations
func (m *MockCentralService) HasAvailableCapacityInRegion(centralRequest *dbapi.CentralRequest) (bool, *errors.ServiceError) {
	return true, nil
}

func (m *MockCentralService) AcceptCentralRequest(centralRequest *dbapi.CentralRequest) *errors.ServiceError {
	return nil
}

func (m *MockCentralService) PrepareCentralRequest(centralRequest *dbapi.CentralRequest) *errors.ServiceError {
	return nil
}

func (m *MockCentralService) Get(ctx context.Context, id string) (*dbapi.CentralRequest, *errors.ServiceError) {
	return nil, nil
}

func (m *MockCentralService) GetByID(id string) (*dbapi.CentralRequest, *errors.ServiceError) {
	return nil, nil
}

func (m *MockCentralService) Delete(centralRequest *dbapi.CentralRequest, force bool) *errors.ServiceError {
	return nil
}

func (m *MockCentralService) RegisterCentralJob(centralRequest *dbapi.CentralRequest) *errors.ServiceError {
	return nil
}

func (m *MockCentralService) List(ctx context.Context, listArgs *services.ListArguments) (dbapi.CentralList, *services.PagingMeta, *errors.ServiceError) {
	return dbapi.CentralList{}, nil, nil
}

func (m *MockCentralService) Create(ctx context.Context, central *dbapi.CentralRequest) (*dbapi.CentralRequest, *errors.ServiceError) {
	return nil, nil
}

func (m *MockCentralService) Update(central *dbapi.CentralRequest) *errors.ServiceError {
	return nil
}

func (m *MockCentralService) UpdateStatus(id string, status dbapi.DataPlaneCentralStatus) *errors.ServiceError {
	return nil
}

func (m *MockCentralService) Deprovision(id string) *errors.ServiceError {
	return nil
}

func (m *MockCentralService) Updates(centralRequest *dbapi.CentralRequest, values map[string]interface{}) *errors.ServiceError {
	return nil
}

func (m *MockCentralService) HasAvailableCapacityInDataPlaneClusters() (bool, *errors.ServiceError) {
	return true, nil
}

func (m *MockCentralService) FindByIDs(ids []string) (dbapi.CentralList, *errors.ServiceError) {
	return dbapi.CentralList{}, nil
}

func (m *MockCentralService) CountByStatus(status []string) ([]services.CentralStatusCount, error) {
	return nil, nil
}

func (m *MockCentralService) ListCentralsToBeDeprovisioned() ([]*dbapi.CentralRequest, error) {
	return nil, nil
}

func (m *MockCentralService) ListCentralsWithExpiredReason(reason string) (dbapi.CentralList, error) {
	return dbapi.CentralList{}, nil
}

// MockManagedCentralPresenter is a mock implementation of presenters.ManagedCentralPresenter
type MockManagedCentralPresenter struct {
	managedCentral *private.ManagedCentral
	err           error
}

func (m *MockManagedCentralPresenter) PresentManagedCentral(central *dbapi.CentralRequest) (*private.ManagedCentral, error) {
	return m.managedCentral, m.err
}

func TestCentralRoutesCNAMEManager_Reconcile_WithUIReachability(t *testing.T) {
	tests := []struct {
		name               string
		central            *dbapi.CentralRequest
		managedCentral     *private.ManagedCentral
		cnameStatus        string
		uiReachable        bool
		uiCheckError       error
		expectRoutesCreated bool
	}{
		{
			name: "UI reachable after CNAME records are in sync",
			central: &dbapi.CentralRequest{
				Meta:             dbapi.Meta{ID: "test-central-1"},
				RoutesCreationID: "change-123",
			},
			managedCentral: &private.ManagedCentral{
				Spec: private.ManagedCentralAllOfSpec{
					UiHost: "test-central-1.example.com",
				},
			},
			cnameStatus:        "INSYNC",
			uiReachable:        true,
			uiCheckError:       nil,
			expectRoutesCreated: true,
		},
		{
			name: "UI not reachable after CNAME records are in sync",
			central: &dbapi.CentralRequest{
				Meta:             dbapi.Meta{ID: "test-central-2"},
				RoutesCreationID: "change-456",
			},
			managedCentral: &private.ManagedCentral{
				Spec: private.ManagedCentralAllOfSpec{
					UiHost: "test-central-2.example.com",
				},
			},
			cnameStatus:        "INSYNC",
			uiReachable:        false,
			uiCheckError:       nil,
			expectRoutesCreated: false,
		},
		{
			name: "UI reachability check fails",
			central: &dbapi.CentralRequest{
				Meta:             dbapi.Meta{ID: "test-central-3"},
				RoutesCreationID: "change-789",
			},
			managedCentral: &private.ManagedCentral{
				Spec: private.ManagedCentralAllOfSpec{
					UiHost: "test-central-3.example.com",
				},
			},
			cnameStatus:        "INSYNC",
			uiReachable:        false,
			uiCheckError:       errors.New("check failed"),
			expectRoutesCreated: false,
		},
		{
			name: "CNAME records not in sync yet",
			central: &dbapi.CentralRequest{
				Meta:             dbapi.Meta{ID: "test-central-4"},
				RoutesCreationID: "change-101",
			},
			managedCentral: &private.ManagedCentral{
				Spec: private.ManagedCentralAllOfSpec{
					UiHost: "test-central-4.example.com",
				},
			},
			cnameStatus:        "PENDING",
			uiReachable:        true,
			uiCheckError:       nil,
			expectRoutesCreated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockCentralService := &MockCentralService{
				centrals: []*dbapi.CentralRequest{tt.central},
				cnameRecordStatus: &route53.GetChangeOutput{
					ChangeInfo: &route53.ChangeInfo{
						Status: aws.String(tt.cnameStatus),
					},
				},
			}

			mockPresenter := &MockManagedCentralPresenter{
				managedCentral: tt.managedCentral,
			}

			mockUIChecker := NewMockUIReachabilityChecker(tt.uiReachable, tt.uiCheckError)

			centralConfig := &config.CentralConfig{
				EnableCentralExternalDomain: true,
			}

			// Create manager with mocks
			manager := &CentralRoutesCNAMEManager{
				centralService:          mockCentralService,
				centralConfig:           centralConfig,
				managedCentralPresenter: mockPresenter,
				uiReachabilityChecker:   mockUIChecker,
			}

			// Run reconciliation
			errs := manager.Reconcile()

			// Verify results
			assert.Empty(t, errs)
			assert.True(t, mockCentralService.updateCalled)
			assert.Equal(t, tt.expectRoutesCreated, tt.central.RoutesCreated)
		})
	}
}

func TestCentralRoutesCNAMEManager_Reconcile_ExternalDNSEnabled(t *testing.T) {
	// Test case where external-dns operator manages the records
	central := &dbapi.CentralRequest{
		Meta: dbapi.Meta{ID: "test-central-externaldns"},
	}

	managedCentral := &private.ManagedCentral{
		Metadata: private.ManagedCentralAllOfMetadata{
			Annotations: map[string]string{
				"external-dns.alpha.kubernetes.io/hostname": "test.example.com",
			},
		},
		Spec: private.ManagedCentralAllOfSpec{
			UiHost: "test-central-externaldns.example.com",
		},
	}

	mockCentralService := &MockCentralService{
		centrals: []*dbapi.CentralRequest{central},
	}

	mockPresenter := &MockManagedCentralPresenter{
		managedCentral: managedCentral,
	}

	mockUIChecker := NewMockUIReachabilityChecker(true, nil)

	centralConfig := &config.CentralConfig{
		EnableCentralExternalDomain: true,
	}

	manager := &CentralRoutesCNAMEManager{
		centralService:          mockCentralService,
		centralConfig:           centralConfig,
		managedCentralPresenter: mockPresenter,
		uiReachabilityChecker:   mockUIChecker,
	}

	errs := manager.Reconcile()

	assert.Empty(t, errs)
	assert.True(t, mockCentralService.updateCalled)
	assert.True(t, central.RoutesCreated)
}

func TestCentralRoutesCNAMEManager_Constructor(t *testing.T) {
	mockCentralService := &MockCentralService{}
	centralConfig := &config.CentralConfig{}
	mockPresenter := &MockManagedCentralPresenter{}

	manager := NewCentralCNAMEManager(mockCentralService, centralConfig, mockPresenter)

	require.NotNil(t, manager)
	assert.NotNil(t, manager.uiReachabilityChecker)
	assert.IsType(t, &HTTPUIReachabilityChecker{}, manager.uiReachabilityChecker)
}

func TestCentralRoutesCNAMEManager_NoUIHost(t *testing.T) {
	// Test case where UI host is empty
	central := &dbapi.CentralRequest{
		Meta:             dbapi.Meta{ID: "test-central-no-host"},
		RoutesCreationID: "change-999",
	}

	managedCentral := &private.ManagedCentral{
		Spec: private.ManagedCentralAllOfSpec{
			UiHost: "", // Empty UI host
		},
	}

	mockCentralService := &MockCentralService{
		centrals: []*dbapi.CentralRequest{central},
		cnameRecordStatus: &route53.GetChangeOutput{
			ChangeInfo: &route53.ChangeInfo{
				Status: aws.String("INSYNC"),
			},
		},
	}

	mockPresenter := &MockManagedCentralPresenter{
		managedCentral: managedCentral,
	}

	mockUIChecker := NewMockUIReachabilityChecker(false, nil)

	centralConfig := &config.CentralConfig{
		EnableCentralExternalDomain: true,
	}

	manager := &CentralRoutesCNAMEManager{
		centralService:          mockCentralService,
		centralConfig:           centralConfig,
		managedCentralPresenter: mockPresenter,
		uiReachabilityChecker:   mockUIChecker,
	}

	errs := manager.Reconcile()

	assert.Empty(t, errs)
	assert.True(t, mockCentralService.updateCalled)
	// Routes should be marked as created even without UI host (DNS records exist)
	assert.True(t, central.RoutesCreated)
}