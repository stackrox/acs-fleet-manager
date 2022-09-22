package common

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	constants2 "github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
)

const (
	defaultCentralReadyTimeout             = 30 * time.Minute
	defaultCentralClusterAssignmentTimeout = 2 * time.Minute
)

// WaitForNumberOfCentralToBeGivenCount - Awaits for the number of dinosaurs to be exactly X
func WaitForNumberOfCentralToBeGivenCount(ctx context.Context, db *db.ConnectionFactory, client *public.APIClient, count int32) error {
	currentCount := int32(-1)
	err := NewPollerBuilder(db).
		IntervalAndTimeout(defaultPollInterval, defaultCentralPollTimeout).
		RetryLogFunction(func(retry int, maxRetry int) string {
			if currentCount == -1 {
				return fmt.Sprintf("Waiting for centrals count to become %d", count)
			}
			return fmt.Sprintf("Waiting for centrals count to become %d (current %d)", count, currentCount)
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
		return fmt.Errorf("waiting for number of centrals: %w", err)
	}
	return nil
}

// WaitForCentralCreateToBeAccepted - Creates a central and awaits for the request to be accepted
func WaitForCentralCreateToBeAccepted(ctx context.Context, db *db.ConnectionFactory, client *public.APIClient, k public.CentralRequestPayload) (central public.CentralRequest, resp *http.Response, err error) {
	currentStatus := ""

	err = NewPollerBuilder(db).
		IntervalAndTimeout(defaultPollInterval, defaultCentralPollTimeout).
		RetryLogFunction(func(retry int, maxRetry int) string {
			if currentStatus == "" {
				return "Waiting for central creation to be accepted"
			}
			return fmt.Sprintf("Waiting for central creation to be accepted (current status %s)", currentStatus)
		}).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			central, resp, err = client.DefaultApi.CreateCentral(ctx, true, k)
			if err != nil {
				return true, fmt.Errorf("waiting for central creation to be accepted: %w", err)
			}
			return resp.StatusCode == http.StatusAccepted, nil
		}).
		Build().Poll()

	if err != nil {
		return central, resp, fmt.Errorf("waiting for central creation to be accepted: %w", err)
	}
	return central, resp, nil

}

// WaitForCentralToReachStatus - Awaits for a dinosaur to reach a specified status
func WaitForCentralToReachStatus(ctx context.Context, db *db.ConnectionFactory, client *public.APIClient, centralID string, status constants2.CentralStatus) (central public.CentralRequest, err error) {
	currentStatus := ""

	glog.Infof("status: " + status.String())

	err = NewPollerBuilder(db).
		IntervalAndTimeout(1*time.Second, defaultCentralReadyTimeout).
		RetryLogFunction(func(retry int, maxRetry int) string {
			if currentStatus == "" {
				return fmt.Sprintf("Waiting for central '%s' to reach status '%s'", centralID, status.String())
			}
			return fmt.Sprintf("Waiting for central '%s' to reach status '%s' (current status %s)", centralID, status.String(), currentStatus)
		}).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			central, _, err = client.DefaultApi.GetCentralById(ctx, centralID)
			if err != nil {
				return true, fmt.Errorf("waiting for central to reach status: %w", err)
			}

			switch central.Status {
			case constants2.CentralRequestStatusFailed.String():
				fallthrough
			case constants2.CentralRequestStatusDeprovision.String():
				fallthrough
			case constants2.CentralRequestStatusDeleting.String():
				return false, errors.Errorf("Waiting for central '%s' to reach status '%s', but status '%s' has been reached instead", centralID, status.String(), central.Status)
			}

			currentStatus = central.Status
			return constants2.CentralStatus(central.Status).CompareTo(status) >= 0, nil
		}).
		Build().Poll()

	if err != nil {
		return central, fmt.Errorf("waiting for central to reach status: %w", err)
	}
	return central, nil
}

// WaitForCentralToBeDeleted - Awaits for a dinosaur to be deleted
func WaitForCentralToBeDeleted(ctx context.Context, db *db.ConnectionFactory, client *public.APIClient, centralID string) error {
	err := NewPollerBuilder(db).
		IntervalAndTimeout(defaultPollInterval, defaultCentralReadyTimeout).
		RetryLogMessagef("Waiting for central '%s' to be deleted", centralID).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			if _, _, err := client.DefaultApi.GetCentralById(ctx, centralID); err != nil {
				if err.Error() == "404 Not Found" {
					return true, nil
				}

				return false, fmt.Errorf("on retrying: %w", err)
			}
			return false, nil
		}).
		Build().Poll()

	if err != nil {
		return fmt.Errorf("waiting for central to be deleted: %w", err)
	}
	return nil
}

// WaitForCentralClusterIDToBeAssigned ...
func WaitForCentralClusterIDToBeAssigned(dbFactory *db.ConnectionFactory, centralRequestName string) (*dbapi.CentralRequest, error) {
	centralFound := &dbapi.CentralRequest{}

	centralErr := NewPollerBuilder(dbFactory).
		IntervalAndTimeout(defaultPollInterval, defaultCentralClusterAssignmentTimeout).
		RetryLogMessagef("Waiting for central named '%s' to have a ClusterID", centralRequestName).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			if err := dbFactory.New().Where("name = ?", centralRequestName).First(centralFound).Error; err != nil {
				return false, err
			}
			glog.Infof("got central instance %v", centralFound)
			return centralFound.ClusterID != "", nil
		}).Build().Poll()

	if centralErr != nil {
		return centralFound, fmt.Errorf("waiting for central cluster ID to be assigned: %w", centralErr)
	}
	return centralFound, nil
}
