package services

import (
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/clusters/types"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/services"
)

//go:generate moq -out cloud_providers_moq.go . CloudProvidersService
type CloudProvidersService interface {
	GetCloudProvidersWithRegions() ([]CloudProviderWithRegions, *errors.ServiceError)
	GetCachedCloudProvidersWithRegions() ([]CloudProviderWithRegions, *errors.ServiceError)
	ListCloudProviders(listArgs *services.ListArguments) ([]api.CloudProvider, *api.PagingMeta, *errors.ServiceError)
	ListCloudProviderRegions(id string, listArgs *services.ListArguments) ([]api.CloudRegion, *api.PagingMeta, *errors.ServiceError)
}

func NewCloudProvidersService() CloudProvidersService {
	return &cloudProvidersService{}
}

type cloudProvidersService struct {
	// TODO: proper implementation following internal/dinosaur/internal/services/cloud_providers.go
}

type CloudProviderWithRegions struct {
	ID         string
	RegionList *types.CloudProviderRegionInfoList
}

type Cluster struct {
	ProviderType api.ClusterProviderType `json:"provider_type"`
}

func (p cloudProvidersService) GetCloudProvidersWithRegions() ([]CloudProviderWithRegions, *errors.ServiceError) {
	return []CloudProviderWithRegions{}, nil 
}

func (p cloudProvidersService) GetCachedCloudProvidersWithRegions() ([]CloudProviderWithRegions, *errors.ServiceError) {
	return []CloudProviderWithRegions{}, nil 
}


func (p cloudProvidersService) ListCloudProviders(listArgs *services.ListArguments) ([]api.CloudProvider, *api.PagingMeta, *errors.ServiceError) {
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}
	return []api.CloudProvider{}, pagingMeta, nil
}

func (p cloudProvidersService) ListCloudProviderRegions(id string, listArgs *services.ListArguments) ([]api.CloudRegion, *api.PagingMeta, *errors.ServiceError) {
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}
	return []api.CloudRegion{}, pagingMeta, nil
}
