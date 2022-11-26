// Package dbprovisioning provides functionality to provision and deprovision databases on cloud providers
package dbprovisioning

import (
	"context"
)

// Client defines an interface for clients that can provision and deprovision databases on cloud providers
type Client interface {
	EnsureDBProvisioned(ctx context.Context, name, passwordSecretName string) (string, error)
	EnsureDBDeprovisioned(name string) (bool, error)
}
