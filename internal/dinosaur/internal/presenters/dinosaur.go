package presenters

import (
	"fmt"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/internal/api/public"
)

// ConvertDinosaurRequest from payload to DinosaurRequest
func ConvertDinosaurRequest(dinosaurRequestPayload public.CentralRequestPayload, dbDinosaurrequest ...*dbapi.DinosaurRequest) *dbapi.DinosaurRequest {
	// TODO implement converter
	var dinosaur = &dbapi.DinosaurRequest{}

	dinosaur.Region = dinosaurRequestPayload.Region
	dinosaur.Name = dinosaurRequestPayload.Name
	dinosaur.CloudProvider = dinosaurRequestPayload.CloudProvider
	dinosaur.MultiAZ = dinosaurRequestPayload.MultiAz

	return dinosaur
}

// PresentDinosaurRequest - create CentralRequest in an appropriate format ready to be returned by the API
func PresentDinosaurRequest(request *dbapi.DinosaurRequest) public.CentralRequest {
	return public.CentralRequest{
		Id: request.ID,
		Kind: "CentralRequest",
		Href: fmt.Sprintf("/api/rhacs/v1/centrals/%s", request.ID),
		Status: request.Status,
		CloudProvider: request.CloudProvider,
		MultiAz: request.MultiAZ,
		Region: request.Region,
		Owner: request.Owner,
		Name: request.Name,
		Host: request.Host,
		CreatedAt: request.CreatedAt,
		UpdatedAt: request.UpdatedAt,
		FailedReason: request.FailedReason,
		Version: "", // TODO(ROX-10709)
		InstanceType: request.InstanceType,
	}
}
