package presenters

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	admin "github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/converters"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/services/account"
)

// PresentDinosaurRequestAdminEndpoint presents a dbapi.CentralRequest as an admin.Dinosaur.
func PresentDinosaurRequestAdminEndpoint(request *dbapi.CentralRequest, _ account.AccountService) (*admin.Central, *errors.ServiceError) {
	var adminCentral admin.CentralSpec
	var adminScanner admin.ScannerSpec

	if len(request.Central) > 0 {
		var central dbapi.CentralSpec
		if err := json.Unmarshal(request.Central, &central); err != nil {
			// Assuming here that what is in the DB is guaranteed to conform to the expected schema.
			glog.Errorf("Failed to unmarshal Central spec %q: %v", request.Central, err)
		}
		adminCentral = admin.CentralSpec{
			Resources: converters.ConvertCoreV1ResourceRequirementsToAdmin(&central.Resources),
		}
	}

	if len(request.Scanner) > 0 {
		var scanner dbapi.ScannerSpec
		if err := json.Unmarshal(request.Scanner, &scanner); err != nil {
			// Assuming here that what is in the DB is guaranteed to conform to the expected schema.
			glog.Errorf("Failed to unmarshal Scanner spec %q: %v", request.Scanner, err)
		}
		adminScanner = admin.ScannerSpec{
			Analyzer: admin.ScannerSpecAnalyzer{
				Resources: converters.ConvertCoreV1ResourceRequirementsToAdmin(&scanner.Analyzer.Resources),
				Scaling:   admin.ScannerSpecAnalyzerScaling(scanner.Analyzer.Scaling),
			},
			Db: admin.ScannerSpecDb{
				Resources: converters.ConvertCoreV1ResourceRequirementsToAdmin(&scanner.Db.Resources),
			},
		}
	}

	return &admin.Central{
		Id:                   request.ID,
		Kind:                 "CentralRequest",
		Href:                 fmt.Sprintf("/api/rhacs/v1/centrals/%s", request.ID),
		Status:               request.Status,
		CloudProvider:        request.CloudProvider,
		MultiAz:              request.MultiAZ,
		Region:               request.Region,
		Owner:                request.Owner,
		Name:                 request.Name,
		Host:                 request.GetUIHost(), // TODO(ROX-11990): Split the Host in Fleet Manager Public API to UI and Data hosts
		CreatedAt:            request.CreatedAt,
		UpdatedAt:            request.UpdatedAt,
		FailedReason:         request.FailedReason,
		ActualCentralVersion: request.ActualCentralVersion,
		InstanceType:         request.InstanceType,
		Central:              adminCentral,
		Scanner:              adminScanner,
	}, nil
}
