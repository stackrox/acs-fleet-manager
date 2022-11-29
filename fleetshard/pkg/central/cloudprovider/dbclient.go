// Package cloudprovider provides cloud-provider specific functionality, such as provisioning of databases
package cloudprovider

import (
	"context"
)

// DBClient defines an interface for clients that can provision and deprovision databases on cloud providers
type DBClient interface {
	// EnsureDBProvisioned is a blocking function that makes sure that an RDS database was provisioned for a Central
	EnsureDBProvisioned(ctx context.Context, name, passwordSecretName string) (string, error)
	EnsureDBDeprovisioned(name string) (bool, error)
}
