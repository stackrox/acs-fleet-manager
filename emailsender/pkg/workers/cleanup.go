package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/emailsender/pkg/db"
)

// CleanupEmailSent is a worker used to periodically cleanup EmailSentByTenant events
// stored in the database connection that are no longer needed to enforce rate limitting
type CleanupEmailSent struct {
	DbConn       db.DatabaseClient
	Period       time.Duration
	ExpiredAfter time.Duration
}

// Run periodically executes a cleanup query against the given DB Connection
func (c *CleanupEmailSent) Run(ctx context.Context) error {
	ticker := time.NewTicker(c.Period)

	glog.Info("Starting CleanupEmailSent worker...")
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return fmt.Errorf("stopped cleanup worker: %w", context.Canceled)
		case <-ticker.C:
			numDeleted, err := c.DbConn.CleanupEmailSentByTenant(time.Now().Add(-c.ExpiredAfter))
			if err != nil {
				glog.Errorf("failed to cleanup EmailSentByTenant: %v", err)
			}

			glog.Infof("deleted %d expired EmailSentByTenant events from DB", numDeleted)
		}
	}

}
