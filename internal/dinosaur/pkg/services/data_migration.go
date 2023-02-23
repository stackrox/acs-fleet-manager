package services

import (
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
)

// DataMigration is the database migration boot service.
// The data migration is executed on boot and is non-blocking. Its purpose is to migrate
// existing data records, which have the `organisation_name` column not yet set.
// Once all records have this column set in production, the data migration service
// becomes obsolete and will be removed again.
// TODO (ROX-15037): remove data migration service.
type DataMigration struct {
	connectionFactory *db.ConnectionFactory
	amsClient         ocm.AMSClient
}

// NewDataMigration creates a new migration service instance.
func NewDataMigration(connectionFactory *db.ConnectionFactory, amsClient ocm.AMSClient) *DataMigration {
	return &DataMigration{connectionFactory: connectionFactory, amsClient: amsClient}
}

// Returns number of migrated records for testing purposes.
func (m *DataMigration) migrateOrganisationNames() (int, error) {
	migratedCnt := 0
	colName := "OrganisationName"
	dbConn := m.connectionFactory.New()
	rows, err := dbConn.Model(&dbapi.CentralRequest{}).Where("COALESCE(organisation_name, '') = ''").Rows()
	if err != nil {
		return migratedCnt, errors.Wrap(err, "querying rows requiring data migration")
	}
	defer func() {
		if err := rows.Close(); err != nil {
			glog.Error(errors.Wrapf(err, "closing cursor in data migration of column %q", colName))
		}
	}()

	for rows.Next() {
		var central dbapi.CentralRequest
		if err := dbConn.ScanRows(rows, &central); err != nil {
			return migratedCnt, errors.Wrap(err, "scanning row record")
		}

		org, err := m.amsClient.GetOrganisationFromExternalID(central.OrganisationID)
		if err != nil {
			return migratedCnt, errors.Wrapf(err, "fetching organisation name to %q from OCM for central instance %q", org.Name(), central.ID)
		}
		if err = dbConn.Model(&central).Update(colName, org.Name()).Error; err != nil {
			return migratedCnt, errors.Wrapf(err, "updating organisation name to %q for central instance %q", org.Name(), central.ID)
		}
		glog.Infof("migrated column %q to new value %q for central instance %q ", colName, central.OrganisationName, central.ID)
		migratedCnt++
	}
	return migratedCnt, nil
}

// Start the migration service.
func (m *DataMigration) Start() {
	_, err := m.migrateOrganisationNames()
	if err != nil {
		glog.Error(errors.Wrapf(err, "data migration of column %q", "organisation_name"))
	}
}

// Stop the migration service.
func (m *DataMigration) Stop() {}
