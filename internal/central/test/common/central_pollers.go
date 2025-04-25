package common

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
)

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
