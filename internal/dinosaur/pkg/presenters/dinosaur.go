package presenters

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
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
	var central public.CentralSpec
	var scanner public.ScannerSpec

	if err := json.Unmarshal(request.Central, &central); err != nil {
		// Assuming here that what is in the DB is guaranteed to conform to the expected schema.
		// TODO: Add error propagation.
		glog.Errorf("Failed to unmarshal Central spec: %v", err)
	}

	if err := json.Unmarshal(request.Scanner, &scanner); err != nil {
		// Assuming here that what is in the DB is guaranteed to conform to the expected schema.
		// TODO: Add error propagation.
		glog.Errorf("Failed to unmarshal Scanner spec: %v", err)
	}

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
		Host:          request.GetUIHost(), // TODO(ROX-11990): Split the Host in Fleet Manager Public API to UI and Data hosts
		CreatedAt:     request.CreatedAt,
		UpdatedAt:     request.UpdatedAt,
		FailedReason:  request.FailedReason,
		Version:       request.ActualCentralVersion,
		InstanceType:  request.InstanceType,
		Central:       central,
		Scanner:       scanner,
	}
}
