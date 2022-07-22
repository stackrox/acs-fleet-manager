package services

import (
	"sort"
	"strings"
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/clusters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/clusters/types"

	"github.com/patrickmn/go-cache"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

const keyCloudProvidersWithRegions = "cloudProviderWithRegions"

var cloudPoviderIDToDisplayNameMapping = map[string]string{
	"aws":   "Amazon Web Services",
	"azure": "Microsoft Azure",
	"gcp":   "Google Cloud Platform",
}

// CloudProvidersService ...
//go:generate moq -out cloud_providers_moq.go . CloudProvidersService
type CloudProvidersService interface {
	GetCloudProvidersWithRegions() ([]CloudProviderWithRegions, *errors.ServiceError)
	GetCachedCloudProvidersWithRegions() ([]CloudProviderWithRegions, *errors.ServiceError)
	ListCloudProviders() ([]api.CloudProvider, *errors.ServiceError)
	ListCloudProviderRegions(id string) ([]api.CloudRegion, *errors.ServiceError)
}

// NewCloudProvidersService ...
func NewCloudProvidersService(providerFactory clusters.ProviderFactory, connectionFactory *db.ConnectionFactory) CloudProvidersService {
	return &cloudProvidersService{
		providerFactory:   providerFactory,
		connectionFactory: connectionFactory,
		cache:             cache.New(5*time.Minute, 10*time.Minute),
	}
}

type cloudProvidersService struct {
	providerFactory   clusters.ProviderFactory
	connectionFactory *db.ConnectionFactory
	cache             *cache.Cache
}

// CloudProviderWithRegions ...
type CloudProviderWithRegions struct {
	ID         string
	RegionList *types.CloudProviderRegionInfoList
}

// Cluster ...
type Cluster struct {
	ProviderType api.ClusterProviderType `json:"provider_type"`
}

// GetCloudProvidersWithRegions ...
func (p cloudProvidersService) GetCloudProvidersWithRegions() ([]CloudProviderWithRegions, *errors.ServiceError) {
	results, dbErr := p.getAvailableClusterProviderTypes()
	if dbErr != nil {
		return nil, dbErr
	}

	cloudProvidersToRegions := map[string]*types.CloudProviderRegionInfoList{}

	for _, result := range results {
		provider, err := p.providerFactory.GetProvider(result.ProviderType)
		if err != nil {
			return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to find implementation")
		}
		providerList, err := provider.GetCloudProviders()
		if err != nil {
			return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to retrieve cloud provider list")
		}
		for _, cp := range providerList.Items {
			regions, regionErr := provider.GetCloudProviderRegions(cp)
			if regionErr != nil {
				return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to retrieve cloud regions")
			}

			existingRegions, ok := cloudProvidersToRegions[cp.ID]
			if !ok {
				cloudProvidersToRegions[cp.ID] = regions
			} else { // merge existing regions with new regions
				existingRegions.Merge(regions)
				cloudProvidersToRegions[cp.ID] = existingRegions
			}
		}

	}

	var cloudProviderWithRegions = []CloudProviderWithRegions{}
	for key, regions := range cloudProvidersToRegions {
		cloudProviderWithRegions = append(cloudProviderWithRegions, CloudProviderWithRegions{
			ID:         key,
			RegionList: regions,
		})
	}

	sort.Slice(cloudProviderWithRegions, func(i, j int) bool {
		return cloudProviderWithRegions[i].ID < cloudProviderWithRegions[j].ID
	})

	return cloudProviderWithRegions, nil
}

// GetCachedCloudProvidersWithRegions ...
func (p cloudProvidersService) GetCachedCloudProvidersWithRegions() ([]CloudProviderWithRegions, *errors.ServiceError) {
	cachedCloudProviderWithRegions, cached := p.cache.Get(keyCloudProvidersWithRegions)
	if cached {
		return convertToCloudProviderWithRegionsType(cachedCloudProviderWithRegions)
	}
	cloudProviderWithRegions, err := p.GetCloudProvidersWithRegions()
	if err != nil {
		return nil, err
	}
	p.cache.Set(keyCloudProvidersWithRegions, cloudProviderWithRegions, cache.DefaultExpiration)
	return cloudProviderWithRegions, nil
}

func convertToCloudProviderWithRegionsType(cachedCloudProviderWithRegions interface{}) ([]CloudProviderWithRegions, *errors.ServiceError) {
	cloudProviderWithRegions, ok := cachedCloudProviderWithRegions.([]CloudProviderWithRegions)
	if ok {
		return cloudProviderWithRegions, nil
	}
	return nil, nil
}

// ListCloudProviders ...
func (p cloudProvidersService) ListCloudProviders() ([]api.CloudProvider, *errors.ServiceError) {
	results, err := p.getAvailableClusterProviderTypes()
	if err != nil {
		return nil, err
	}

	alreadyVisitedCloudProviders := map[string]bool{}
	cloudProviderList := []api.CloudProvider{}
	for _, result := range results {
		provider, err := p.providerFactory.GetProvider(result.ProviderType)
		if err != nil {
			return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to find implementation")
		}
		providerList, err := provider.GetCloudProviders()
		if err != nil {
			return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to retrieve cloud provider list")
		}
		for _, cp := range providerList.Items {
			_, cloudProviderAlreadyCollected := alreadyVisitedCloudProviders[cp.ID]
			if cloudProviderAlreadyCollected {
				continue
			}
			cloudProviderList = append(cloudProviderList, api.CloudProvider{
				ID:          cp.ID,
				Name:        cp.Name,
				DisplayName: setDisplayName(cp.ID, cp.DisplayName),
			})
			alreadyVisitedCloudProviders[cp.ID] = true
		}
	}

	return cloudProviderList, nil
}

// ListCloudProviderRegions ...
func (p cloudProvidersService) ListCloudProviderRegions(id string) ([]api.CloudRegion, *errors.ServiceError) {
	cloudRegionList := []api.CloudRegion{}
	cloudProviders, err := p.GetCloudProvidersWithRegions()
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to retrieve cloud provider regions")
	}

	for _, cloudProvider := range cloudProviders {
		if cloudProvider.ID == id {
			for _, r := range cloudProvider.RegionList.Items {
				cloudRegionList = append(cloudRegionList, api.CloudRegion{
					ID:            r.ID,
					CloudProvider: r.CloudProviderID,
					DisplayName:   r.DisplayName,
				})
			}
			break
		}
	}

	return cloudRegionList, nil
}

func (p cloudProvidersService) getAvailableClusterProviderTypes() ([]Cluster, *errors.ServiceError) {
	dbConn := p.connectionFactory.New().
		Model(&Cluster{}).
		Distinct("provider_type").
		Where("status NOT IN (?)", api.ClusterDeletionStatuses)

	var results []Cluster
	err := dbConn.Find(&results).Error
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Failed to list clusters providers")
	}

	return results, nil
}

func setDisplayName(providerID string, defaultDisplayName string) string {
	displayName, ok := cloudPoviderIDToDisplayNameMapping[strings.ToLower(providerID)]
	if ok {
		return displayName
	}

	return defaultDisplayName
}
