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
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
	coreServices "github.com/stackrox/acs-fleet-manager/pkg/services"
)

// ValidDinosaurClusterNameRegexp ...
var ValidDinosaurClusterNameRegexp = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

// MaxDinosaurNameLength ...
var MaxDinosaurNameLength = 32

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
func ValidateCentralSpec(ctx context.Context, centralRequestPayload *public.CentralRequestPayload, field string, dbCentral *dbapi.CentralRequest) handlers.Validate {
	return func() *errors.ServiceError {
		// Validate Central resources.
		if err := validateQuantity(centralRequestPayload.Central.Resources.Requests.Cpu, "central.resources.requests.cpu"); err != nil {
			return err
		}
		if err := validateQuantity(centralRequestPayload.Central.Resources.Requests.Memory, "central.resources.requests.memory"); err != nil {
			return err
		}
		if err := validateQuantity(centralRequestPayload.Central.Resources.Limits.Cpu, "central.resources.limits.cpu"); err != nil {
			return err
		}
		if err := validateQuantity(centralRequestPayload.Central.Resources.Limits.Cpu, "central.resources.limits.memory"); err != nil {
			return err
		}
		central, err := json.Marshal(centralRequestPayload.Central)
		if err != nil {
			return errors.Validation("marshaling Central spec failed: %v", err)
		}

		dbCentral.Central = central
		return nil
	}
}

// ValidateScannerSpec ...
func ValidateScannerSpec(ctx context.Context, centralRequestPayload *public.CentralRequestPayload, field string, dbCentral *dbapi.CentralRequest) handlers.Validate {
	return func() *errors.ServiceError {
		// Validate Scanner Analyzer resources and scaling settings.
		if err := validateQuantity(centralRequestPayload.Scanner.Analyzer.Resources.Requests.Cpu, "scanner.analyzer.resources.requests.cpu"); err != nil {
			return err
		}
		if err := validateQuantity(centralRequestPayload.Scanner.Analyzer.Resources.Requests.Memory, "scanner.analyzer.resources.requests.memory"); err != nil {
			return err
		}
		if err := validateQuantity(centralRequestPayload.Scanner.Analyzer.Resources.Limits.Cpu, "scanner.analyzer.resources.limits.cpu"); err != nil {
			return err
		}
		if err := validateQuantity(centralRequestPayload.Scanner.Analyzer.Resources.Limits.Cpu, "scanner.analyzer.resources.limits.memory"); err != nil {
			return err
		}
		if centralRequestPayload.Scanner.Analyzer.Scaling.AutoScaling != "" &&
			centralRequestPayload.Scanner.Analyzer.Scaling.AutoScaling != "Enabled" &&
			centralRequestPayload.Scanner.Analyzer.Scaling.AutoScaling != "Disabled" {
			return errors.Validation("invalid AutoScaling setting at Scanner.Analyzer.Scaling.AutoScaling, expected 'Enabled' or 'Disabled'")
		}

		// Validate Scanner DB resources.
		if err := validateQuantity(centralRequestPayload.Scanner.Db.Resources.Requests.Cpu, "scanner.db.resources.requests.cpu"); err != nil {
			return err
		}
		if err := validateQuantity(centralRequestPayload.Scanner.Db.Resources.Requests.Memory, "scanner.db.resources.requests.memory"); err != nil {
			return err
		}
		if err := validateQuantity(centralRequestPayload.Scanner.Db.Resources.Limits.Cpu, "scanner.db.resources.limits.cpu"); err != nil {
			return err
		}
		if err := validateQuantity(centralRequestPayload.Scanner.Db.Resources.Limits.Cpu, "scanner.db.resources.limits.memory"); err != nil {
			return err
		}

		scanner, err := json.Marshal(centralRequestPayload.Scanner)
		if err != nil {
			return errors.Validation("marshaling Scanner spec failed: %v", err)
		}

		dbCentral.Scanner = scanner
		return nil
	}
}
