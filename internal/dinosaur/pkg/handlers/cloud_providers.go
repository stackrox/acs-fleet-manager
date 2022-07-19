package handlers

import (
	"net/http"
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"

	"github.com/patrickmn/go-cache"

	"github.com/gorilla/mux"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

const cloudProvidersCacheKey = "cloudProviderList"

type cloudProvidersHandler struct {
	service            services.CloudProvidersService
	cache              *cache.Cache
	supportedProviders config.ProviderList
}

func NewCloudProviderHandler(service services.CloudProvidersService, providerConfig *config.ProviderConfig) *cloudProvidersHandler {
	return &cloudProvidersHandler{
		service:            service,
		supportedProviders: providerConfig.ProvidersConfig.SupportedProviders,
		cache:              cache.New(5*time.Minute, 10*time.Minute),
	}
}

func (h cloudProvidersHandler) ListCloudProviderRegions(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	query := r.URL.Query()
	instanceTypeFilter := query.Get("instance_type")
	cacheId := id
	if instanceTypeFilter != "" {
		cacheId = cacheId + "-" + instanceTypeFilter
	}

	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.ValidateLength(&id, "id", &handlers.MinRequiredFieldLength, nil),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			cachedRegionList, cached := h.cache.Get(cacheId)
			if cached {
				return cachedRegionList, nil
			}
			cloudRegions, err := h.service.ListCloudProviderRegions(id)
			if err != nil {
				return nil, err
			}
			regionList := public.CloudRegionList{
				Kind:  "CloudRegionList",
				Page:  int32(1),
				Items: []public.CloudRegion{},
			}

			provider, _ := h.supportedProviders.GetByName(id)
			for i := range cloudRegions {
				cloudRegion := cloudRegions[i]
				region, _ := provider.Regions.GetByName(cloudRegion.Id)

				// skip any regions that do not support the specified instance type so its not included in the response
				if instanceTypeFilter != "" && !region.IsInstanceTypeSupported(config.InstanceType(instanceTypeFilter)) {
					continue
				}

				// Only set enabled to true if the region supports at least one instance type
				cloudRegion.Enabled = len(region.SupportedInstanceTypes) > 0
				cloudRegion.SupportedInstanceTypes = region.SupportedInstanceTypes.AsSlice()
				converted := presenters.PresentCloudRegion(&cloudRegion)
				regionList.Items = append(regionList.Items, converted)
			}

			regionList.Total = int32(len(regionList.Items))
			regionList.Size = int32(len(regionList.Items))

			h.cache.Set(cacheId, regionList, cache.DefaultExpiration)
			return regionList, nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h cloudProvidersHandler) ListCloudProviders(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			cachedCloudProviderList, cached := h.cache.Get(cloudProvidersCacheKey)
			if cached {
				return cachedCloudProviderList, nil
			}
			cloudProviders, err := h.service.ListCloudProviders()
			if err != nil {
				return nil, err
			}
			cloudProviderList := public.CloudProviderList{
				Kind:  "CloudProviderList",
				Total: int32(len(cloudProviders)),
				Size:  int32(len(cloudProviders)),
				Page:  int32(1),
				Items: []public.CloudProvider{},
			}

			for i := range cloudProviders {
				_, cloudProviders[i].Enabled = h.supportedProviders.GetByName(cloudProviders[i].Id)
				converted := presenters.PresentCloudProvider(&cloudProviders[i])
				cloudProviderList.Items = append(cloudProviderList.Items, converted)
			}
			h.cache.Set(cloudProvidersCacheKey, cloudProviderList, cache.DefaultExpiration)
			return cloudProviderList, nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}
