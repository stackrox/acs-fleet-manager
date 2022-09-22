package presenters

import (
	"fmt"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/public"
)

// ConvertCentralRequest from payload to CentralRequest
func ConvertCentralRequest(centralRequestPayload public.CentralRequestPayload, dbCentralRequest ...*dbapi.CentralRequest) *dbapi.CentralRequest {
	// TODO implement converter
	var central = &dbapi.CentralRequest{}

	central.Region = centralRequestPayload.Region
	central.Name = centralRequestPayload.Name
	central.CloudProvider = centralRequestPayload.CloudProvider
	central.MultiAZ = centralRequestPayload.MultiAz

	return central
}

// PresentCentralRequest - create CentralRequest in an appropriate format ready to be returned by the API
func PresentCentralRequest(request *dbapi.CentralRequest) public.CentralRequest {
	return public.CentralRequest{
		Id:             request.ID,
		Kind:           "CentralRequest",
		Href:           fmt.Sprintf("/api/rhacs/v1/centrals/%s", request.ID),
		Status:         request.Status,
		CloudProvider:  request.CloudProvider,
		MultiAz:        request.MultiAZ,
		Region:         request.Region,
		Owner:          request.Owner,
		Name:           request.Name,
		CentralUIURL:   fmt.Sprintf("https://%s", request.GetUIHost()),
		CentralDataURL: fmt.Sprintf("https://%s", request.GetDataHost()),
		CreatedAt:      request.CreatedAt,
		UpdatedAt:      request.UpdatedAt,
		FailedReason:   request.FailedReason,
		Version:        request.ActualCentralVersion,
		InstanceType:   request.InstanceType,
	}
}
