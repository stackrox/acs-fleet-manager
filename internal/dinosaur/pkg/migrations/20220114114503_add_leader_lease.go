package migrations

import (
	"fmt"
	"time"

	"github.com/stackrox/acs-fleet-manager/pkg/db"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"gorm.io/gorm"
)

// LeaderLease ...
type LeaderLease struct {
	db.Model
	Leader    string
	LeaseType string
	Expires   *time.Time
}

var dinosaurLeaseTypes = []string{"accepted_dinosaur", "preparing_dinosaur", "provisioning_dinosaur", "deleting_dinosaur", "ready_dinosaur", "dinosaur_dns", "general_dinosaur_worker"}
var clusterLeaseTypes = []string{"cluster"}

// addLeaderLease adds the LeaderLease data type and adds some leader lease values
// intended to belong to the the Dinousar and Cluster data type worker types
func addLeaderLease() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20220114114503",
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&LeaderLease{})
			if err != nil {
				return fmt.Errorf("auto migrating: %w", err)
			}

			// Set an initial already expired lease
			clusterLeaseExpireTime := time.Now().Add(1 * time.Minute)
			for _, leaderLeaseType := range clusterLeaseTypes {
				if err := tx.Create(&api.LeaderLease{Expires: &clusterLeaseExpireTime, LeaseType: leaderLeaseType, Leader: api.NewID()}).Error; err != nil {
					return err
				}
			}

			for _, leaderLeaseType := range dinosaurLeaseTypes {
				if err := tx.Create(&api.LeaderLease{Expires: &db.DinosaurAdditionalLeasesExpireTime, LeaseType: leaderLeaseType, Leader: api.NewID()}).Error; err != nil {
					return err
				}
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Migrator().DropTable(&LeaderLease{})
			if err != nil {
				return fmt.Errorf("rolling back: %w", err)
			}
			return nil
		},
	}
}
