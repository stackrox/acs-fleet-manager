package presenters

import (
	"fmt"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
)

const (
	sensorDataPort = 443 // The port for connecting sensor to the data URL.
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

// PresentCentralRequest - create CentralRequest in an appropriate format ready to be returned by the API
func PresentCentralRequest(request *dbapi.CentralRequest) public.CentralRequest {
	outputRequest := public.CentralRequest{
		Id:             request.ID,
		Kind:           "CentralRequest",
		Href:           fmt.Sprintf("/api/rhacs/v1/centrals/%s", request.ID),
		Status:         request.Status,
		CloudProvider:  request.CloudProvider,
		CloudAccountId: request.CloudAccountID,
		MultiAz:        request.MultiAZ,
		Region:         request.Region,
		Owner:          request.Owner,
		Name:           request.Name,
		CreatedAt:      request.CreatedAt,
		UpdatedAt:      request.UpdatedAt,
		FailedReason:   request.FailedReason,
		Version:        request.ActualCentralVersion,
		InstanceType:   request.InstanceType,
	}

	if request.RoutesCreated {
		if request.GetUIHost() != "" {
			outputRequest.CentralUIURL = fmt.Sprintf("https://%s", request.GetUIHost())
		}
		if request.GetDataHost() != "" {
			outputRequest.CentralDataURL = fmt.Sprintf("%s:%d", request.GetDataHost(), sensorDataPort)
		}
	}

	return outputRequest
}
