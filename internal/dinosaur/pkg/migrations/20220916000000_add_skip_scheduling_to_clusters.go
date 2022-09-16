package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"gorm.io/gorm"
)

func addSkipSchedulingToClusters() *gormigrate.Migration {
	type Cluster struct {
		db.Model
		CloudProvider                    string   `json:"cloud_provider"`
		ClusterID                        string   `json:"cluster_id" gorm:"uniqueIndex:uix_clusters_cluster_id"`
		ExternalID                       string   `json:"external_id"`
		MultiAZ                          bool     `json:"multi_az"`
		Region                           string   `json:"region"`
		Status                           string   `json:"status" gorm:"index"`
		StatusDetails                    string   `json:"status_details" gorm:"-"`
		IdentityProviderID               string   `json:"identity_provider_id"`
		ClusterDNS                       string   `json:"cluster_dns"`
		ProviderType                     string   `json:"provider_type"`
		ProviderSpec                     string   `json:"provider_spec"`
		ClusterSpec                      string   `json:"cluster_spec"`
		AvailableCentralOperatorVersions api.JSON `json:"available_central_operator_versions"`
		SupportedInstanceType            string   `json:"supported_instance_type"`
		SkipScheduling                   bool     `json:"skip_scheduling" gorm:"default:false"`
	}

	return &gormigrate.Migration{
		ID: "20220916000000",
		Migrate: func(tx *gorm.DB) error {
			err := tx.AutoMigrate(&Cluster{})
			if err != nil {
				return fmt.Errorf("migrating 20220114114501: %w", err)
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			err := tx.Migrator().DropTable(&Cluster{})
			if err != nil {
				return fmt.Errorf("rolling back 20220114114501: %w", err)
			}
			return nil
		},
	}
}
