package migrations

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"gorm.io/gorm"
)

var oldLeaseTypes = []string{"accepted_dinosaur", "preparing_dinosaur", "provisioning_dinosaur", "deleting_dinosaur", "ready_dinosaur", "dinosaur_dns", "general_dinosaur_worker"}
var newLeaseTypes = []string{"accepted_central", "preparing_central", "provisioning_central", "deleting_central", "ready_central", "central_dns", "general_central_worker"}

func renameLeaderLeaseTypes() *gormigrate.Migration {

	return &gormigrate.Migration{
		ID: "20250428100000",
		Migrate: func(tx *gorm.DB) error {
			// Update only the central requests which do not already specify client_origin after migrating the table
			// schema. For central requests which do not specify it, we can assume shared_static_sso, since the field
			// was not available before adding the possibility to create centrals with a dynamic OIDC client.
			// **Note**: This update is _not_ done within a transaction since we pass `UseTransaction=false` as
			// gormigrate option, meaning this will be on a best-effort basis, but the risk of not doing it within a
			// transaction is minor.
			for n := range oldLeaseTypes {
				old := oldLeaseTypes[n]
				new := newLeaseTypes[n]
				err := tx.Model(&api.LeaderLease{}).
					Where("lease_type = ?", old).
					Update("lease_type", new).Error
				if err != nil {
					return fmt.Errorf("renaming lease_type from %s to %s in "+
						"LeaderLease in migration 20250428100000: %w", old, new, err)
				}
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			for n := range oldLeaseTypes {
				err := tx.Model(&api.LeaderLease{}).
					Where("lease_type = ?", newLeaseTypes[n]).
					Update("lease_type", oldLeaseTypes[n]).Error
				if err != nil {
					return fmt.Errorf("rollback renaming lease_types in "+
						"LeaderLease in migration 20250428100000: %w", err)
				}
			}
			return nil
		},
	}
}
