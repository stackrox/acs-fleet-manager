// Package cloudprovider provides cloud-provider specific functionality, such as provisioning of databases
package cloudprovider

import (
	"context"
)

// DBClient defines an interface for clients that can provision and deprovision databases on cloud providers
//
//go:generate moq -out dbclient_moq.go . DBClient
type DBClient interface {
	// EnsureDBProvisioned is a blocking function that makes sure that a database with the given databaseID was provisioned,
	// using the master password given as parameter
	EnsureDBProvisioned(ctx context.Context, databaseID, passwordSecretName string) (string, error)
	// EnsureDBDeprovisioned is a non-blocking function that makes sure that a managed DB is deprovisioned (more
	// specifically, that its deletion was initiated)
	EnsureDBDeprovisioned(databaseID string) (bool, error)
}
