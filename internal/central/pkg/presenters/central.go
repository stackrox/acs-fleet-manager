package presenters

import (
	"fmt"

	"github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/public"
)

const (
	sensorDataPort = 443 // The port for connecting sensor to the data URL.
)

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
		InstanceType:   request.InstanceType,
	}

	if request.Status == constants.CentralRequestStatusReady.String() && request.RoutesCreated {
		if request.GetUIHost() != "" {
			outputRequest.CentralUIURL = fmt.Sprintf("https://%s", request.GetUIHost())
		}
		if request.GetDataHost() != "" {
			outputRequest.CentralDataURL = fmt.Sprintf("%s:%d", request.GetDataHost(), sensorDataPort)
		}
	}

	return outputRequest
}
