package presenters

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/public"
)

// ConvertDinosaurRequest from payload to DinosaurRequest
func ConvertDinosaurRequest(dinosaurRequestPayload public.DinosaurRequestPayload, dbDinosaurrequest ...*dbapi.DinosaurRequest) *dbapi.DinosaurRequest {
	// TODO implement converter
	var dinosaur = &dbapi.DinosaurRequest{}

	dinosaur.Region = dinosaurRequestPayload.Region
	dinosaur.Name = dinosaurRequestPayload.Name
	dinosaur.CloudProvider = dinosaurRequestPayload.CloudProvider
	dinosaur.MultiAZ = dinosaurRequestPayload.MultiAz

	return dinosaur
}

// PresentDinosaurRequest - create DinosaurRequest in an appropriate format ready to be returned by the API
func PresentDinosaurRequest(dinosaurRequest *dbapi.DinosaurRequest) public.DinosaurRequest {
	var res public.DinosaurRequest
	res.Name = dinosaurRequest.Name
	res.Status = dinosaurRequest.Status
	res.Host = dinosaurRequest.Host
	res.Region = dinosaurRequest.Region
	res.CreatedAt = dinosaurRequest.CreatedAt
	res.CloudProvider = dinosaurRequest.CloudProvider
	res.FailedReason = dinosaurRequest.FailedReason
	res.Status = dinosaurRequest.Status
	res.InstanceType = dinosaurRequest.InstanceType
	res.Id = dinosaurRequest.ID
	res.MultiAz = dinosaurRequest.MultiAZ
	res.Version = dinosaurRequest.ActualDinosaurVersion
	res.Owner = dinosaurRequest.Owner

	return res
}
