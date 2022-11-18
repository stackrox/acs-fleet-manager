package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
	coreServices "github.com/stackrox/acs-fleet-manager/pkg/services"
)

var (
	// ValidDinosaurClusterNameRegexp ...
	ValidDinosaurClusterNameRegexp = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

	// MaxDinosaurNameLength ...
	MaxDinosaurNameLength = 32

	supportedResources = []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory}
)

// ValidDinosaurClusterName ...
func ValidDinosaurClusterName(value *string, field string) handlers.Validate {
	return func() *errors.ServiceError {
		if !ValidDinosaurClusterNameRegexp.MatchString(*value) {
			return errors.MalformedDinosaurClusterName("%s does not match %s", field, ValidDinosaurClusterNameRegexp.String())
		}
		return nil
	}
}

// ValidateDinosaurClusterNameIsUnique returns a validator that validates that the dinosaur cluster name is unique
func ValidateDinosaurClusterNameIsUnique(context context.Context, name *string, dinosaurService services.DinosaurService) handlers.Validate {
	return func() *errors.ServiceError {

		_, pageMeta, err := dinosaurService.List(context, &coreServices.ListArguments{Page: 1, Size: 1, Search: fmt.Sprintf("name = %s", *name)})
		if err != nil {
			return err
		}

		if pageMeta.Total > 0 {
			return errors.DuplicateDinosaurClusterName()
		}

		return nil
	}
}

// ValidateCloudProvider returns a validator that sets default cloud provider details if needed and validates provided
// provider and region
func ValidateCloudProvider(dinosaurService *services.DinosaurService, dinosaurRequest *dbapi.CentralRequest, providerConfig *config.ProviderConfig, action string) handlers.Validate {
	return func() *errors.ServiceError {
		// Set Cloud Provider default if not received in the request
		supportedProviders := providerConfig.ProvidersConfig.SupportedProviders
		if dinosaurRequest.CloudProvider == "" {
			defaultProvider, _ := supportedProviders.GetDefault()
			dinosaurRequest.CloudProvider = defaultProvider.Name
		}

		// Validation for Cloud Provider
		provider, providerSupported := supportedProviders.GetByName(dinosaurRequest.CloudProvider)
		if !providerSupported {
			return errors.ProviderNotSupported("provider %s is not supported, supported providers are: %s", dinosaurRequest.CloudProvider, supportedProviders)
		}

		// Set Cloud Region default if not received in the request
		if dinosaurRequest.Region == "" {
			defaultRegion, _ := provider.GetDefaultRegion()
			dinosaurRequest.Region = defaultRegion.Name
		}

		// Validation for Cloud Region
		regionSupported := provider.IsRegionSupported(dinosaurRequest.Region)
		if !regionSupported {
			return errors.RegionNotSupported("region %s is not supported for %s, supported regions are: %s", dinosaurRequest.Region, dinosaurRequest.CloudProvider, provider.Regions)
		}

		// Validate Region/InstanceType
		instanceType, err := (*dinosaurService).DetectInstanceType(dinosaurRequest)
		if err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, "error detecting instance type: %s", err.Error())
		}

		region, _ := provider.Regions.GetByName(dinosaurRequest.Region)
		if !region.IsInstanceTypeSupported(config.InstanceType(instanceType)) {
			return errors.InstanceTypeNotSupported("instance type '%s' not supported for region '%s'", instanceType.String(), region.Name)
		}
		return nil
	}
}

// ValidateDinosaurClaims ...
func ValidateDinosaurClaims(ctx context.Context, dinosaurRequestPayload *public.CentralRequestPayload, dinosaurRequest *dbapi.CentralRequest) handlers.Validate {
	return func() *errors.ServiceError {
		dinosaurRequest.Region = dinosaurRequestPayload.Region
		dinosaurRequest.Name = dinosaurRequestPayload.Name
		dinosaurRequest.CloudProvider = dinosaurRequestPayload.CloudProvider
		dinosaurRequest.MultiAZ = dinosaurRequestPayload.MultiAz
		dinosaurRequest.CloudAccountID = dinosaurRequestPayload.CloudAccountId

		claims, err := auth.GetClaimsFromContext(ctx)
		if err != nil {
			return errors.Unauthenticated("user not authenticated")
		}

		dinosaurRequest.Owner, _ = claims.GetUsername()
		dinosaurRequest.OrganisationID, _ = claims.GetOrgID()
		dinosaurRequest.OwnerAccountID, _ = claims.GetAccountID()
		dinosaurRequest.OwnerUserID, _ = claims.GetUserID()

		return nil
	}
}

func validateQuantity(qty string, path string) *errors.ServiceError {
	if qty == "" {
		return nil
	}
	_, err := resource.ParseQuantity(qty)
	if err != nil {
		return errors.Validation("invalid resources: failed to parse quantity %q at %s due to: %v", qty, path, err)
	}
	return nil
}

// ValidateCentralSpec ...
func ValidateCentralSpec(ctx context.Context, centralRequestPayload *public.CentralRequestPayload, dbCentral *dbapi.CentralRequest) handlers.Validate {
	return func() *errors.ServiceError {
		// Validate Central resources.
		err := validateResourceList(centralRequestPayload.Central.Resources.Requests, "central.resources.requests")
		if err != nil {
			return errors.Validation("invalid resource requests for Central: %v", err)
		}
		err = validateResourceList(centralRequestPayload.Central.Resources.Limits, "central.resources.limits")
		if err != nil {
			return errors.Validation("invalid resource limits for Central: %v", err)
		}

		central, err := json.Marshal(centralRequestPayload.Central)
		if err != nil {
			return errors.Validation("marshaling Central spec failed: %v", err)
		}

		if err := json.Unmarshal(central, &dbapi.CentralSpec{}); err != nil {
			return errors.Validation("invalid value as Central spec: %v", err)
		}

		dbCentral.Central = central
		return nil
	}
}

func validateResourceList(resources map[string]string, path string) error {
	for name, qty := range resources {
		resourceName := corev1.ResourceName(name)
		if resourceName != corev1.ResourceCPU && resourceName != corev1.ResourceMemory {
			return errors.Validation("unsupported resource type %q in %s", name, path)
		}
		if qty == "" {
			continue
		}
		_, err := resource.ParseQuantity(qty)
		if err != nil {
			return errors.Validation("invalid resources: failed to parse quantity %q at %s.%s due to: %v", qty, path, name, err)
		}
	}
	return nil
}

// ValidateScannerSpec ...
func ValidateScannerSpec(ctx context.Context, centralRequestPayload *public.CentralRequestPayload, dbCentral *dbapi.CentralRequest) handlers.Validate {
	return func() *errors.ServiceError {
		// Validate Scanner Analyzer resources and scaling settings.
		err := validateResourceList(centralRequestPayload.Scanner.Analyzer.Resources.Requests, "scanner.analyzer.resources.requests")
		if err != nil {
			return errors.Validation("invalid resource requests for Scanner Analyzer: %v", err)
		}
		err = validateResourceList(centralRequestPayload.Scanner.Analyzer.Resources.Limits, "scanner.analyzer.resources.limits")
		if err != nil {
			return errors.Validation("invalid resource limits for Scanner Analyzer: %v", err)
		}

		if centralRequestPayload.Scanner.Analyzer.Scaling.AutoScaling != "" &&
			centralRequestPayload.Scanner.Analyzer.Scaling.AutoScaling != "Enabled" &&
			centralRequestPayload.Scanner.Analyzer.Scaling.AutoScaling != "Disabled" {
			return errors.Validation("invalid AutoScaling setting at Scanner.Analyzer.Scaling.AutoScaling, expected 'Enabled' or 'Disabled'")
		}

		// Validate Scanner DB resources.
		err = validateResourceList(centralRequestPayload.Scanner.Db.Resources.Requests, "scanner.db.resources.requests")
		if err != nil {
			return errors.Validation("invalid resource requests for Scanner DB: %v", err)
		}
		err = validateResourceList(centralRequestPayload.Scanner.Analyzer.Resources.Limits, "scanner.db.resources.limits")
		if err != nil {
			return errors.Validation("invalid resource limits for Scanner DB: %v", err)
		}

		// Marshal ScannerSpec into byte string.
		scanner, err := json.Marshal(centralRequestPayload.Scanner)
		if err != nil {
			return errors.Validation("marshaling Scanner spec failed: %v", err)
		}

		if err := json.Unmarshal(scanner, &dbapi.ScannerSpec{}); err != nil {
			return errors.Validation("invalid value as Scanner spec: %v", err)
		}

		dbCentral.Scanner = scanner
		return nil
	}
}

// ValidateScannerAnalyzerScaling validates the provided Scanner Analyzer Scaling configuration.
func ValidateScannerAnalyzerScaling(scaling *dbapi.ScannerAnalyzerScaling) error {
	if scaling == nil {
		return nil
	}

	if scaling.AutoScaling != "Enabled" && scaling.AutoScaling != "Disabled" {
		return fmt.Errorf("invalid scaling configuration: unknown AutoScaling %q, must be 'Enabled' or 'Disabled'", scaling.AutoScaling)
	}
	if scaling.MinReplicas <= 0 {
		return fmt.Errorf("invalid scaling configuration: MinReplicas (%v) must be positive", scaling.MinReplicas)
	}
	if scaling.Replicas <= 0 {
		return fmt.Errorf("invalid scaling configuration: Replicas (%v) must be positive", scaling.Replicas)
	}
	if scaling.MaxReplicas <= 0 {
		return fmt.Errorf("invalid scaling configuration: MaxReplicas (%v) must be positive", scaling.MaxReplicas)
	}
	if scaling.Replicas < scaling.MinReplicas {
		return fmt.Errorf("invalid scaling configuration: Replicas (%v) < MinReplicas (%v)", scaling.Replicas, scaling.MinReplicas)
	}
	if scaling.Replicas > scaling.MaxReplicas {
		return fmt.Errorf("invalid scaling configuration: Replicas (%v) > MaxReplicas (%v)", scaling.Replicas, scaling.MaxReplicas)
	}

	return nil
}

// ValidateResourceName checks if the given name denotes a supported resource.
func ValidateResourceName(name string) (corev1.ResourceName, bool) {
	resourceName := corev1.ResourceName(name)
	for _, supportedResource := range supportedResources {
		if supportedResource == resourceName {
			return resourceName, true
		}
	}
	return corev1.ResourceName(""), false
}
