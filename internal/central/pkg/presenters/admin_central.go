// Package presenters ...
package presenters

import (
	"fmt"

	admin "github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/services/account"
)

// PresentCentralRequestAdminEndpoint presents a dbapi.CentralRequest as an admin.Central.
func PresentCentralRequestAdminEndpoint(request *dbapi.CentralRequest, _ account.AccountService) (*admin.Central, *errors.ServiceError) {
	return &admin.Central{
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
		ExpiredAt:     dbapi.NullTimeToTimePtr(request.ExpiredAt),
		FailedReason:  request.FailedReason,
		InstanceType:  request.InstanceType,
		Traits:        request.Traits,
		ClusterId:     request.ClusterID,
	}, nil
}
