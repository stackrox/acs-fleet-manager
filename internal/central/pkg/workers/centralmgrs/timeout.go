package centralmgrs

import (
	"time"

	"github.com/pkg/errors"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
)

// FailIfTimeoutExceeded checks timeout on a central instance and moves it to failed if timeout is exceeded.
// Returns true if timeout is exceeded, otherwise false.
func FailIfTimeoutExceeded(centralService services.CentralService, timeout time.Duration, centralRequest *dbapi.CentralRequest) error {
	referencePoint := centralRequest.CreatedAt
	if centralRequest.EnteredProvisioningAt.Valid {
		referencePoint = centralRequest.EnteredProvisioningAt.Time
	}

	if referencePoint.Before(time.Now().Add(-timeout)) {
		centralRequest.Status = constants2.CentralRequestStatusFailed.String()
		centralRequest.FailedReason = "Creation time went over the timeout. Interrupting central initialization."

		if err := centralService.UpdateIgnoreNils(centralRequest); err != nil {
			return errors.Wrapf(err, "failed to update timed out central %s", centralRequest.ID)
		}
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants2.CentralRequestStatusFailed, centralRequest.ID, centralRequest.ClusterID, time.Since(centralRequest.CreatedAt))
		metrics.IncreaseCentralTimeoutCountMetric(centralRequest.ID, centralRequest.ClusterID)
		return errors.Errorf("Central request timed out: %s", centralRequest.ID)
	}
	return nil
}
