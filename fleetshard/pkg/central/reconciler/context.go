package reconciler

import (
	"context"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
)

// managedCentralKey is the key used to store the managed central in a context.
// it is populated at the very beginning by CentralReconciler and available to all reconcilers.
type managedCentralKey struct{}

func withManagedCentral(ctx context.Context, central private.ManagedCentral) context.Context {
	return context.WithValue(ctx, managedCentralKey{}, central)
}

func managedCentralFromContext(ctx context.Context) (private.ManagedCentral, bool) {
	central, ok := ctx.Value(managedCentralKey{}).(private.ManagedCentral)
	return central, ok
}
