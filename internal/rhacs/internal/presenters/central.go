package presenters

import (
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/api/public"
)

const (
	CentralRequestListKind = "CentralRequestList"
)

// ConvertCentralRequest from payload to CentralRequest
func ConvertCentralRequest(requestPayload public.CentralRequestPayload, dbRequest ...*dbapi.CentralRequest) *dbapi.CentralRequest {
	// TODO implement converter
	var request *dbapi.CentralRequest = &dbapi.CentralRequest{}

	request.Region = requestPayload.Region
	request.Name = requestPayload.Name
	request.CloudProvider = requestPayload.CloudProvider
	request.MultiAZ = requestPayload.MultiAz

	return request
}

// PresentCentralRequest - create CentralRequest in an appropriate format ready to be returned by the API
func PresentCentralRequest(request *dbapi.CentralRequest) public.CentralRequest {
	// TODO implement presenter
	var res public.CentralRequest

	return res
}