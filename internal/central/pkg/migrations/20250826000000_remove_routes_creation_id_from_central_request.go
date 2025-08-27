package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"gorm.io/gorm"
)

func removeRoutesCreationIDFromCentralRequest() *gormigrate.Migration {
	type CentralRequest struct {
		RoutesCreationID string `json:"routes_creation_id"`
		RoutesCreated    bool   `json:"routes_created"`
	}

	return &gormigrate.Migration{
		ID: "20250826000000",
		Migrate: func(tx *gorm.DB) error {
			// Remove routes_creation_id and routes_created columns from central_requests table
			// since Route53 record management has been moved to external DNS
			if err := dropIfColumnExists(tx, &dbapi.CentralRequest{}, "routes_creation_id"); err != nil {
				return err
			}
			return dropIfColumnExists(tx, &dbapi.CentralRequest{}, "routes_created")
		},
		Rollback: func(tx *gorm.DB) error {
			// Re-add the columns on rollback
			if err := addColumnIfNotExists(tx, &CentralRequest{}, "routes_creation_id"); err != nil {
				return err
			}
			return addColumnIfNotExists(tx, &CentralRequest{}, "routes_created")
		},
	}
}
