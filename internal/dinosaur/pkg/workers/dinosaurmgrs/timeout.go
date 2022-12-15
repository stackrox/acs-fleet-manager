package dinosaurmgrs

import (
	"time"

	"github.com/pkg/errors"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
)

// FailIfTimeoutExceeded checks timeout on a dinosaur instance and moves it to failed if timeout is exceeded.
func FailIfTimeoutExceeded(dinosaurService services.DinosaurService, timeout time.Duration, dinosaur *dbapi.CentralRequest) error {
	if dinosaur.CreatedAt.Before(time.Now().Add(-timeout)) {
		dinosaur.Status = constants2.CentralRequestStatusFailed.String()
		dinosaur.FailedReason = "Creation time went over the timeout. Interrupting central initialization."

		if err := dinosaurService.Update(dinosaur); err != nil {
			return errors.Wrapf(err, "failed to update timed out central %s", dinosaur.ID)
		}
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants2.CentralRequestStatusFailed, dinosaur.ID, dinosaur.ClusterID, time.Since(dinosaur.CreatedAt))
		metrics.IncreaseCentralTimeoutCountMetric(dinosaur.ID, dinosaur.ClusterID)
	}
	return nil
}
