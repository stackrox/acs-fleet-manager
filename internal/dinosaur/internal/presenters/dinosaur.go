package presenters

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/public"
)

// ConvertDinosaurRequest from payload to DinosaurRequest
func ConvertDinosaurRequest(dinosaurRequestPayload public.CentralRequestPayload, dbDinosaurrequest ...*dbapi.DinosaurRequest) *dbapi.DinosaurRequest {
	// TODO implement converter
	var dinosaur *dbapi.DinosaurRequest = &dbapi.DinosaurRequest{}

	dinosaur.Region = dinosaurRequestPayload.Region
	dinosaur.Name = dinosaurRequestPayload.Name
	dinosaur.CloudProvider = dinosaurRequestPayload.CloudProvider
	dinosaur.MultiAZ = dinosaurRequestPayload.MultiAz

	return dinosaur
}

// PresentDinosaurRequest - create CentralRequest in an appropriate format ready to be returned by the API
func PresentDinosaurRequest(dinosaurRequest *dbapi.DinosaurRequest) public.CentralRequest {
	// TODO implement presenter
	var res public.CentralRequest

	return res
}
