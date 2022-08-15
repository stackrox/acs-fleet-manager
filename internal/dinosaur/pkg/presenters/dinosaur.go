package presenters

import (
	"fmt"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
)

// ConvertDinosaurRequest from payload to DinosaurRequest
func ConvertDinosaurRequest(dinosaurRequestPayload public.CentralRequestPayload, dbDinosaurrequest ...*dbapi.CentralRequest) *dbapi.CentralRequest {
	// TODO implement converter
	var dinosaur = &dbapi.CentralRequest{}

	dinosaur.Region = dinosaurRequestPayload.Region
	dinosaur.Name = dinosaurRequestPayload.Name
	dinosaur.CloudProvider = dinosaurRequestPayload.CloudProvider
	dinosaur.MultiAZ = dinosaurRequestPayload.MultiAz

	return dinosaur
}

// PresentDinosaurRequest - create CentralRequest in an appropriate format ready to be returned by the API
func PresentDinosaurRequest(request *dbapi.CentralRequest) public.CentralRequest {
	return public.CentralRequest{
		Id:            request.ID,
		Kind:          "CentralRequest",
		Href:          fmt.Sprintf("/api/rhacs/v1/centrals/%s", request.ID),
		Status:        request.Status,
		CloudProvider: request.CloudProvider,
		MultiAz:       request.MultiAZ,
		Region:        request.Region,
		Owner:         request.Owner,
		Name:          request.Name,
		UiHost:        request.GetUIHost(),
		DataHost:      request.GetDataHost(),
		CreatedAt:     request.CreatedAt,
		UpdatedAt:     request.UpdatedAt,
		FailedReason:  request.FailedReason,
		Version:       request.ActualCentralVersion,
		InstanceType:  request.InstanceType,
	}
}
