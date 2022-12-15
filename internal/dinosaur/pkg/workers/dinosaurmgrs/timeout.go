package dinosaurmgrs

import (
	"time"

	"github.com/pkg/errors"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
)

// FailIfTimeoutExceeded checks timeout on a central instance and moves it to failed if timeout is exceeded.
// Returns true if timeout is exceeded, otherwise false.
func FailIfTimeoutExceeded(centralService services.DinosaurService, timeout time.Duration, centralRequest *dbapi.CentralRequest) error {
	if centralRequest.CreatedAt.Before(time.Now().Add(-timeout)) {
		centralRequest.Status = constants2.CentralRequestStatusFailed.String()
		centralRequest.FailedReason = "Creation time went over the timeout. Interrupting central initialization."

		if err := centralService.Update(centralRequest); err != nil {
			return errors.Wrapf(err, "failed to update timed out central %s", centralRequest.ID)
		}
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants2.CentralRequestStatusFailed, centralRequest.ID, centralRequest.ClusterID, time.Since(centralRequest.CreatedAt))
		metrics.IncreaseCentralTimeoutCountMetric(centralRequest.ID, centralRequest.ClusterID)
		return errors.Errorf("Central request timed out: %s", centralRequest.ID)
	}
	return nil
}
