package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/public"
)

// ConvertDinosaurRequest from payload to DinosaurRequest
func ConvertDinosaurRequest(dinosaurRequestPayload public.DinosaurRequestPayload, dbDinosaurrequest ...*dbapi.DinosaurRequest) *dbapi.DinosaurRequest {
	// TODO implement converter
	var dinosaur *dbapi.DinosaurRequest = &dbapi.DinosaurRequest{}

	dinosaur.Region = dinosaurRequestPayload.Region
	dinosaur.Name = dinosaurRequestPayload.Name
	dinosaur.CloudProvider = dinosaurRequestPayload.CloudProvider
	dinosaur.MultiAZ = dinosaurRequestPayload.MultiAz

	return dinosaur
}

// PresentDinosaurRequest - create DinosaurRequest in an appropriate format ready to be returned by the API
func PresentDinosaurRequest(dinosaurRequest *dbapi.DinosaurRequest) public.DinosaurRequest {
	// TODO implement presenter
	var res public.DinosaurRequest

	return res
}
