package common

import (
	"context"
	"fmt"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"net/http"
)

// WaitForDinosaurCreateToBeAccepted - Creates a dinosaur and awaits for the request to be accepted
func WaitForDinosaurCreateToBeAccepted(ctx context.Context, db *db.ConnectionFactory, client *public.APIClient, k public.CentralRequestPayload) (dinosaur public.CentralRequest, resp *http.Response, err error) {
	currentStatus := ""

	err = NewPollerBuilder(db).
		IntervalAndTimeout(defaultPollInterval, defaultDinosaurPollTimeout).
		RetryLogFunction(func(retry int, maxRetry int) string {
			if currentStatus == "" {
				return "Waiting for central creation to be accepted"
			}
			return fmt.Sprintf("Waiting for central creation to be accepted (current status %s)", currentStatus)
		}).
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			dinosaur, resp, err = client.DefaultApi.CreateCentral(ctx, true, k)
			if err != nil {
				return true, fmt.Errorf("waiting for central creation to be accepted: %w", err)
			}
			return resp.StatusCode == http.StatusAccepted, nil
		}).
		Build().Poll()

	if err != nil {
		return dinosaur, resp, fmt.Errorf("waiting for central creation to be accepted: %w", err)
	}
	return dinosaur, resp, nil

}
