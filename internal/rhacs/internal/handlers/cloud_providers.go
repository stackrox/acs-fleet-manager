package handlers

import (
	"net/http"
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/config"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/services"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
	coreServices "github.com/stackrox/acs-fleet-manager/pkg/services"

	"github.com/patrickmn/go-cache"

	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/gorilla/mux"
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
			listArgs := coreServices.NewListArguments(r.URL.Query())
			cloudRegions, paging, err := h.service.ListCloudProviderRegions(id, listArgs)
			if err != nil {
				return nil, err
			}
			regionList := public.CloudRegionList{
				Kind:  "CloudRegionList",
				Size:  int32(paging.Size),
				Page:  int32(paging.Page),
				Items: []public.CloudRegion{},
			}

			provider, _ := h.supportedProviders.GetByName(id)
			for _, cloudRegion := range cloudRegions {
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
			listArgs := coreServices.NewListArguments(r.URL.Query())
			cloudProviders, paging, err := h.service.ListCloudProviders(listArgs)
			if err != nil {
				return nil, err
			}
			cloudProviderList := public.CloudProviderList{
				Kind:  "CloudProviderList",
				Size:  int32(paging.Size),
				Page:  int32(paging.Page),
				Items: []public.CloudProvider{},
			}

			for _, cloudProvider := range cloudProviders {
				_, cloudProvider.Enabled = h.supportedProviders.GetByName(cloudProvider.Id)
				converted := presenters.PresentCloudProvider(&cloudProvider)
				cloudProviderList.Items = append(cloudProviderList.Items, converted)
			}
			h.cache.Set(cloudProvidersCacheKey, cloudProviderList, cache.DefaultExpiration)
			return cloudProviderList, nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}
