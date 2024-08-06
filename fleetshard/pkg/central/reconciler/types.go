package reconciler

import "context"

type reconciler interface {
	ensurePresent(ctx context.Context) (context.Context, error)
	ensureAbsent(ctx context.Context) (context.Context, error)
}

type noopReconciler struct{}

func (noopReconciler) ensurePresent(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

func (noopReconciler) ensureAbsent(ctx context.Context) (context.Context, error) {
	return ctx, nil
}
