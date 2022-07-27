package common

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
)

const (
	defaultDinosaurReadyTimeout             = 30 * time.Minute
	defaultDinosaurClusterAssignmentTimeout = 2 * time.Minute
)

// WaitForNumberOfDinosaurToBeGivenCount - Awaits for the number of dinosaurs to be exactly X
func WaitForNumberOfDinosaurToBeGivenCount(ctx context.Context, db *db.ConnectionFactory, client *public.APIClient, count int32) error {
	currentCount := int32(-1)
	err := NewPollerBuilder(db).
		IntervalAndTimeout(defaultPollInterval, defaultDinosaurPollTimeout).
		RetryLogFunction(func(retry int, maxRetry int) string {
			if currentCount == -1 {
				return fmt.Sprintf("Waiting for dinosaurs count to become %d", count)
			}
			return fmt.Sprintf("Waiting for dinosaurs count to become %d (current %d)", count, currentCount)
		}).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			list, _, err := client.DefaultApi.GetCentrals(ctx, nil)
			if err != nil {
				return false, fmt.Errorf("retrying: %w", err)
			}
			currentCount = list.Size
			return currentCount == count, nil
		}).
		Build().Poll()

	if err != nil {
		return fmt.Errorf("waiting for number of dinosaurs: %w", err)
	}
	return nil
}

// WaitForDinosaurCreateToBeAccepted - Creates a dinosaur and awaits for the request to be accepted
func WaitForDinosaurCreateToBeAccepted(ctx context.Context, db *db.ConnectionFactory, client *public.APIClient, k public.CentralRequestPayload) (dinosaur public.CentralRequest, resp *http.Response, err error) {
	currentStatus := ""

	err = NewPollerBuilder(db).
		IntervalAndTimeout(defaultPollInterval, defaultDinosaurPollTimeout).
		RetryLogFunction(func(retry int, maxRetry int) string {
			if currentStatus == "" {
				return "Waiting for dinosaur creation to be accepted"
			}
			return fmt.Sprintf("Waiting for dinosaur creation to be accepted (current status %s)", currentStatus)
		}).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			dinosaur, resp, err = client.DefaultApi.CreateCentral(ctx, true, k)
			if err != nil {
				return true, fmt.Errorf("waiting for dinosaur creation to be accepted: %w", err)
			}
			return resp.StatusCode == http.StatusAccepted, nil
		}).
		Build().Poll()

	if err != nil {
		return dinosaur, nil, fmt.Errorf("waiting for dinosaur creation to be accepted: %w", err)
	}
	return dinosaur, resp, nil

}

// WaitForDinosaurToReachStatus - Awaits for a dinosaur to reach a specified status
func WaitForDinosaurToReachStatus(ctx context.Context, db *db.ConnectionFactory, client *public.APIClient, dinosaurID string, status constants2.DinosaurStatus) (dinosaur public.CentralRequest, err error) {
	currentStatus := ""

	glog.Infof("status: " + status.String())

	err = NewPollerBuilder(db).
		IntervalAndTimeout(1*time.Second, defaultDinosaurReadyTimeout).
		RetryLogFunction(func(retry int, maxRetry int) string {
			if currentStatus == "" {
				return fmt.Sprintf("Waiting for dinosaur '%s' to reach status '%s'", dinosaurID, status.String())
			}
			return fmt.Sprintf("Waiting for dinosaur '%s' to reach status '%s' (current status %s)", dinosaurID, status.String(), currentStatus)
		}).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			dinosaur, _, err = client.DefaultApi.GetCentralById(ctx, dinosaurID)
			if err != nil {
				return true, fmt.Errorf("waiting for dinosaur to reach status: %w", err)
			}

			switch dinosaur.Status {
			case constants2.DinosaurRequestStatusFailed.String():
				fallthrough
			case constants2.DinosaurRequestStatusDeprovision.String():
				fallthrough
			case constants2.DinosaurRequestStatusDeleting.String():
				return false, errors.Errorf("Waiting for dinosaur '%s' to reach status '%s', but status '%s' has been reached instead", dinosaurID, status.String(), dinosaur.Status)
			}

			currentStatus = dinosaur.Status
			return constants2.DinosaurStatus(dinosaur.Status).CompareTo(status) >= 0, nil
		}).
		Build().Poll()

	if err != nil {
		return dinosaur, fmt.Errorf("waiting for dinosaur to reach status: %w", err)
	}
	return dinosaur, nil
}

// WaitForDinosaurToBeDeleted - Awaits for a dinosaur to be deleted
func WaitForDinosaurToBeDeleted(ctx context.Context, db *db.ConnectionFactory, client *public.APIClient, dinosaurID string) error {
	err := NewPollerBuilder(db).
		IntervalAndTimeout(defaultPollInterval, defaultDinosaurReadyTimeout).
		RetryLogMessagef("Waiting for dinosaur '%s' to be deleted", dinosaurID).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			if _, _, err := client.DefaultApi.GetCentralById(ctx, dinosaurID); err != nil {
				if err.Error() == "404 Not Found" {
					return true, nil
				}

				return false, fmt.Errorf("on retrying: %w", err)
			}
			return false, nil
		}).
		Build().Poll()

	if err != nil {
		return fmt.Errorf("waiting for dinosaur to be deleted: %w", err)
	}
	return nil
}

// WaitForDinosaurClusterIDToBeAssigned ...
func WaitForDinosaurClusterIDToBeAssigned(dbFactory *db.ConnectionFactory, dinosaurRequestName string) (*dbapi.CentralRequest, error) {
	dinosaurFound := &dbapi.CentralRequest{}

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

	if dinosaurErr != nil {
		return dinosaurFound, fmt.Errorf("waiting for dinosaur cluster ID to be assigned: %w", dinosaurErr)
	}
	return dinosaurFound, nil
}
