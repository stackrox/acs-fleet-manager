package presenters
/* FIXME implement analogon for rhacs

import (
	"fmt"

	compat "github.com/stackrox/acs-fleet-manager/internal/dinosaur/dinosaur_compat"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/handlers"
)

const (
	// TODO change kind to correspond to your own service
	// KindDinosaur is a string identifier for the type api.DinosaurRequest
	KindDinosaur = "Dinosaur"
	// CloudRegion is a string identifier for the type api.CloudRegion
	KindCloudRegion = "CloudRegion"
	// KindCloudProvider is a string identifier for the type api.CloudProvider
	KindCloudProvider = "CloudProvider"
	// KindError is a string identifier for the type api.ServiceError
	KindError = "Error"

	// TODO change base path to correspond to your service
	BasePath = "/api/dinosaurs_mgmt/v1"
)

func PresentReference(id, obj interface{}) compat.ObjectReference {
	return handlers.PresentReferenceWith(id, obj, objectKind, objectPath)
}

func objectKind(i interface{}) string {
	switch i.(type) {
	case dbapi.DinosaurRequest, *dbapi.DinosaurRequest:
		return KindDinosaur
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
	case dbapi.DinosaurRequest, *dbapi.DinosaurRequest:
		return fmt.Sprintf("%s/dinosaurs/%s", BasePath, id) // TODO change /dinosaurs to match your rest resource
	case errors.ServiceError, *errors.ServiceError:
		return fmt.Sprintf("%s/errors/%s", BasePath, id)
	default:
		return ""
	}
}
*/