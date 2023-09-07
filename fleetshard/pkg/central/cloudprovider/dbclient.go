// Package cloudprovider provides cloud-provider specific functionality, such as provisioning of databases
package cloudprovider

import (
	"context"
	"errors"

	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/postgres"
)

// DBClient defines an interface for clients that can provision and deprovision databases on cloud providers
//
//go:generate moq -out dbclient_moq.go . DBClient
type DBClient interface {
	// EnsureDBProvisioned is a blocking function that makes sure that a database with the given databaseID was provisioned,
	// using the master password given as parameter
	EnsureDBProvisioned(ctx context.Context, databaseID, acsInstanceID, passwordSecretName string, isTestInstance bool) error
	// EnsureDBDeprovisioned is a non-blocking function that makes sure that a managed DB is deprovisioned (more
	// specifically, that its deletion was initiated)
	EnsureDBDeprovisioned(databaseID string, skipFinalSnapshot bool) error
	// GetDBConnection returns a postgres.DBConnection struct, which contains the data necessary
	// to construct a PostgreSQL connection string. It expects that the database was already provisioned.
	GetDBConnection(databaseID string) (postgres.DBConnection, error)
	// GetAccountQuotas returns database-related service quotas for the cloud provider region on which
	// the instance of fleetshard-sync runs
	GetAccountQuotas(ctx context.Context) (AccountQuotas, error)
}

// AccountQuotas maps a service to its quota values
type AccountQuotas map[AccountQuotaType]AccountQuotaValue

// AccountQuotaType uniquely identifies an account quota type
type AccountQuotaType int

// Database account quota types that are relevant to fleetshard-sync
const (
	DBClusters AccountQuotaType = iota
	DBInstances
	DBSnapshots
)

// ErrDBBackupInProgress is returned if an action failed because a DB backup is in progress
var ErrDBBackupInProgress = errors.New("DB Backup in Progress")

// ErrDBNotFound is returned if an action failed because a expected DB is not found
var ErrDBNotFound = errors.New("DB not found")

// AccountQuotaValue holds quota data for services, as a pair of currently Used out of Max
type AccountQuotaValue struct {
	Used int64
	Max  int64
}
