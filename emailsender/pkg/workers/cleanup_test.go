package workers

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/acs-fleet-manager/emailsender/pkg/db"
	"github.com/stretchr/testify/require"
)

var testTimeout = time.Second * 10

func TestCleanupEmailSent(t *testing.T) {
	mockDB := &db.MockDatabaseClient{
		CleanupEmailSentByTenantFunc: func(before time.Time) (int64, error) { return 5, nil },
	}

	cleanup := &CleanupEmailSent{
		Period:       time.Second * 1,
		ExpiredAfter: time.Hour * 48,
		DbConn:       mockDB,
	}

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*3)
	defer cancel()

	timeoutTimer := time.NewTimer(testTimeout)
	defer timeoutTimer.Stop()

	var errChannel = make(chan error)

	go func() {
		err := cleanup.Run(ctx)
		errChannel <- err
	}()

	select {
	case err := <-errChannel:
		// Expect DB cleanup to be called at least once since this has been running for 3 seconds
		// until the context gets canceled
		require.True(t, mockDB.CalledCleanupEmailSentByTenant, "expected db cleanup to be called, but was not")
		require.ErrorIs(t, err, context.Canceled)
	case <-timeoutTimer.C:
		t.Fatal("cleanup did not stop on canceled context")
	}

}
