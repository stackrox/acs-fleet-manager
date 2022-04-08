package migrations

// Migrations should NEVER use types from other packages. Types can change
// and then migrations run on a _new_ database will fail or behave unexpectedly.
// Instead of importing types, always re-create the type in the migration, as
// is done here, even though the same type is defined in pkg/api

import (
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func addClusters() *gormigrate.Migration {
	type Cluster struct {
		db.Model
		CloudProvider                     string   `json:"cloud_provider"`
		ClusterID                         string   `json:"cluster_id" gorm:"uniqueIndex:uix_clusters_cluster_id"`
		ExternalID                        string   `json:"external_id"`
		MultiAZ                           bool     `json:"multi_az"`
		Region                            string   `json:"region"`
		Status                            string   `json:"status" gorm:"index"`
		StatusDetails                     string   `json:"status_details" gorm:"-"`
		IdentityProviderID                string   `json:"identity_provider_id"`
		ClusterDNS                        string   `json:"cluster_dns"`
		ProviderType                      string   `json:"provider_type"`
		ProviderSpec                      string   `json:"provider_spec"`
		ClusterSpec                       string   `json:"cluster_spec"`
		AvailableDinosaurOperatorVersions api.JSON `json:"available_dinosaur_operator_versions"`
		SupportedInstanceType             string   `json:"supported_instance_type"`
	}

	return &gormigrate.Migration{
		ID: "20220114114501",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Cluster{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&Cluster{})
		},
	}
}
