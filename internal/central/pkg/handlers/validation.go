package handlers

import (
	"context"
	"fmt"
	"regexp"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
	coreServices "github.com/stackrox/acs-fleet-manager/pkg/services"
	genericvalidation "k8s.io/apimachinery/pkg/api/validation"
)

var (
	// ValidCentralClusterNameRegexp ...
	ValidCentralClusterNameRegexp = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

	// MaxCentralNameLength ...
	MaxCentralNameLength = 32
)

// ValidCentralClusterName ...
func ValidCentralClusterName(value *string, field string) handlers.Validate {
	return func() *errors.ServiceError {
		if errs := genericvalidation.NameIsDNSSubdomain(*value, false); len(errs) > 0 {
			return errors.MalformedCentralClusterName("%s is invalid: %v", field, errs)
		}
		if !ValidCentralClusterNameRegexp.MatchString(*value) {
			return errors.MalformedCentralClusterName("%s does not match %s", field, ValidCentralClusterNameRegexp.String())
		}
		return nil
	}
}

// ValidateCentralClusterNameIsUnique returns a validator that validates that the central cluster name is unique
func ValidateCentralClusterNameIsUnique(context context.Context, name *string, centralService services.CentralService) handlers.Validate {
	return func() *errors.ServiceError {

		_, pageMeta, err := centralService.List(context, &coreServices.ListArguments{Page: 1, Size: 1, Search: fmt.Sprintf("name = %s", *name)})
		if err != nil {
			return err
		}

		if pageMeta.Total > 0 {
			return errors.DuplicateCentralClusterName()
		}

		return nil
	}
}

// ValidateCloudProvider returns a validator that sets default cloud provider details if needed and validates provided
// provider and region
func ValidateCloudProvider(centralService *services.CentralService, centralRequest *dbapi.CentralRequest, providerConfig *config.ProviderConfig, action string) handlers.Validate {
	return func() *errors.ServiceError {
		// Set Cloud Provider default if not received in the request
		supportedProviders := providerConfig.ProvidersConfig.SupportedProviders
		if centralRequest.CloudProvider == "" {
			defaultProvider, _ := supportedProviders.GetDefault()
			centralRequest.CloudProvider = defaultProvider.Name
		}

		// Validation for Cloud Provider
		provider, providerSupported := supportedProviders.GetByName(centralRequest.CloudProvider)
		if !providerSupported {
			return errors.ProviderNotSupported("provider %s is not supported, supported providers are: %s", centralRequest.CloudProvider, supportedProviders)
		}

		// Set Cloud Region default if not received in the request
		if centralRequest.Region == "" {
			defaultRegion, _ := provider.GetDefaultRegion()
			centralRequest.Region = defaultRegion.Name
		}

		// Validation for Cloud Region
		regionSupported := provider.IsRegionSupported(centralRequest.Region)
		if !regionSupported {
			return errors.RegionNotSupported("region %s is not supported for %s, supported regions are: %s", centralRequest.Region, centralRequest.CloudProvider, provider.Regions)
		}

		// Validate Region/InstanceType
		instanceType := (*centralService).DetectInstanceType(centralRequest)
		region, _ := provider.Regions.GetByName(centralRequest.Region)
		if !region.IsInstanceTypeSupported(config.InstanceType(instanceType)) {
			return errors.InstanceTypeNotSupported("instance type '%s' not supported for region '%s'", instanceType.String(), region.Name)
		}
		return nil
	}
}

// ValidateCentralClaims ...
func ValidateCentralClaims(ctx context.Context, centralRequestPayload *public.CentralRequestPayload, centralRequest *dbapi.CentralRequest) handlers.Validate {
	return func() *errors.ServiceError {
		centralRequest.Region = centralRequestPayload.Region
		centralRequest.Name = centralRequestPayload.Name
		centralRequest.CloudProvider = centralRequestPayload.CloudProvider
		centralRequest.MultiAZ = centralRequestPayload.MultiAz
		centralRequest.CloudAccountID = centralRequestPayload.CloudAccountId

		claims, err := auth.GetClaimsFromContext(ctx)
		if err != nil {
			return errors.Unauthenticated("user not authenticated")
		}

		centralRequest.Owner, _ = claims.GetUsername()
		centralRequest.OrganisationID, _ = claims.GetOrgID()
		centralRequest.OwnerAccountID, _ = claims.GetAccountID()
		centralRequest.OwnerUserID, _ = claims.GetSubject()
		centralRequest.OwnerAlternateUserID, _ = claims.GetAlternateUserID()

		return nil
	}
}
