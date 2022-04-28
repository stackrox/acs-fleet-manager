package presenters

import (
	"fmt"

	compat "github.com/stackrox/acs-fleet-manager/internal/rhacs/compat"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
)

const (
	// KindCentral is a string identifier for the type api.CentralRequest
	KindCentral = "Central"
	// CloudRegion is a string identifier for the type api.CloudRegion
	KindCloudRegion = "CloudRegion"
	// KindCloudProvider is a string identifier for the type api.CloudProvider
	KindCloudProvider = "CloudProvider"
	// KindError is a string identifier for the type api.ServiceError
	KindError = "Error"

	// TODO change base path to correspond to your service
	BasePath = "/api/rhacs/v1"
)

func PresentReference(id, obj interface{}) compat.ObjectReference {
	return handlers.PresentReferenceWith(id, obj, objectKind, objectPath)
}

func objectKind(i interface{}) string {
	switch i.(type) {
	case dbapi.CentralRequest, *dbapi.CentralRequest:
		return KindCentral
	case api.CloudRegion, *api.CloudRegion:
		return KindCloudRegion
	case api.CloudProvider, *api.CloudProvider:
		return KindCloudProvider
	case errors.ServiceError, *errors.ServiceError:
		return KindError
	default:
		return ""
	}
}

func objectPath(id string, obj interface{}) string {
	switch obj.(type) {
	case dbapi.CentralRequest, *dbapi.CentralRequest:
		return fmt.Sprintf("%s/centrals/%s", BasePath, id)
	case errors.ServiceError, *errors.ServiceError:
		return fmt.Sprintf("%s/errors/%s", BasePath, id)
	default:
		return ""
	}
}
