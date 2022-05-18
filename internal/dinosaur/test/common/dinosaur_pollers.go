package common

import (
	"context"
	"fmt"
	"net/http"
	"time"

	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

const (
	defaultDinosaurReadyTimeout             = 30 * time.Minute
	defaultDinosaurClusterAssignmentTimeout = 2 * time.Minute
)

// WaitForNumberOfDinosaurToBeGivenCount - Awaits for the number of dinosaurs to be exactly X
func WaitForNumberOfDinosaurToBeGivenCount(ctx context.Context, db *db.ConnectionFactory, client *public.APIClient, count int32) error {
	currentCount := int32(-1)

	return NewPollerBuilder(db).
		IntervalAndTimeout(defaultPollInterval, defaultDinosaurPollTimeout).
		RetryLogFunction(func(retry int, maxRetry int) string {
			if currentCount == -1 {
				return fmt.Sprintf("Waiting for dinosaurs count to become %d", count)
			} else {
				return fmt.Sprintf("Waiting for dinosaurs count to become %d (current %d)", count, currentCount)
			}
		}).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			if list, _, err := client.DefaultApi.GetCentrals(ctx, nil); err != nil {
				return false, err
			} else {
				currentCount = list.Size
				return currentCount == count, err
			}
		}).
		Build().Poll()
}

// WaitForDinosaurCreateToBeAccepted - Creates a dinosaur and awaits for the request to be accepted
func WaitForDinosaurCreateToBeAccepted(ctx context.Context, db *db.ConnectionFactory, client *public.APIClient, k public.CentralRequestPayload) (dinosaur public.CentralRequest, resp *http.Response, err error) {
	currentStatus := ""

	err = NewPollerBuilder(db).
		IntervalAndTimeout(defaultPollInterval, defaultDinosaurPollTimeout).
		RetryLogFunction(func(retry int, maxRetry int) string {
			if currentStatus == "" {
				return "Waiting for dinosaur creation to be accepted"
			} else {
				return fmt.Sprintf("Waiting for dinosaur creation to be accepted (current status %s)", currentStatus)
			}
		}).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			dinosaur, resp, err = client.DefaultApi.CreateCentral(ctx, true, k)
			if err != nil {
				return true, err
			}
			return resp.StatusCode == http.StatusAccepted, err
		}).
		Build().Poll()
	return dinosaur, resp, err
}

// WaitForDinosaurToReachStatus - Awaits for a dinosaur to reach a specified status
func WaitForDinosaurToReachStatus(ctx context.Context, db *db.ConnectionFactory, client *public.APIClient, dinosaurId string, status constants2.DinosaurStatus) (dinosaur public.CentralRequest, err error) {
	currentStatus := ""

	glog.Infof("status: " + status.String())

	err = NewPollerBuilder(db).
		IntervalAndTimeout(1*time.Second, defaultDinosaurReadyTimeout).
		RetryLogFunction(func(retry int, maxRetry int) string {
			if currentStatus == "" {
				return fmt.Sprintf("Waiting for dinosaur '%s' to reach status '%s'", dinosaurId, status.String())
			} else {
				return fmt.Sprintf("Waiting for dinosaur '%s' to reach status '%s' (current status %s)", dinosaurId, status.String(), currentStatus)
			}
		}).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			dinosaur, _, err = client.DefaultApi.GetCentralById(ctx, dinosaurId)
			if err != nil {
				return true, err
			}

			switch dinosaur.Status {
			case constants2.DinosaurRequestStatusFailed.String():
				fallthrough
			case constants2.DinosaurRequestStatusDeprovision.String():
				fallthrough
			case constants2.DinosaurRequestStatusDeleting.String():
				return false, errors.Errorf("Waiting for dinosaur '%s' to reach status '%s', but status '%s' has been reached instead", dinosaurId, status.String(), dinosaur.Status)
			}

			currentStatus = dinosaur.Status
			return constants2.DinosaurStatus(dinosaur.Status).CompareTo(status) >= 0, nil
		}).
		Build().Poll()
	return dinosaur, err
}

// WaitForDinosaurToBeDeleted - Awaits for a dinosaur to be deleted
func WaitForDinosaurToBeDeleted(ctx context.Context, db *db.ConnectionFactory, client *public.APIClient, dinosaurId string) error {
	return NewPollerBuilder(db).
		IntervalAndTimeout(defaultPollInterval, defaultDinosaurReadyTimeout).
		RetryLogMessagef("Waiting for dinosaur '%s' to be deleted", dinosaurId).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			if _, _, err := client.DefaultApi.GetCentralById(ctx, dinosaurId); err != nil {
				if err.Error() == "404 Not Found" {
					return true, nil
				}

				return false, err
			}
			return false, nil
		}).
		Build().Poll()
}

func WaitForDinosaurClusterIDToBeAssigned(dbFactory *db.ConnectionFactory, dinosaurRequestName string) (*dbapi.DinosaurRequest, error) {
	dinosaurFound := &dbapi.DinosaurRequest{}

	dinosaurErr := NewPollerBuilder(dbFactory).
		IntervalAndTimeout(defaultPollInterval, defaultDinosaurClusterAssignmentTimeout).
		RetryLogMessagef("Waiting for dinosaur named '%s' to have a ClusterID", dinosaurRequestName).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			if err := dbFactory.New().Where("name = ?", dinosaurRequestName).First(dinosaurFound).Error; err != nil {
				return false, err
			}
			glog.Infof("got dinosaur instance %v", dinosaurFound)
			return dinosaurFound.ClusterID != "", nil
		}).Build().Poll()

	return dinosaurFound, dinosaurErr
}
