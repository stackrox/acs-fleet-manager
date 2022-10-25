package reconciler

import "github.com/pkg/errors"

var (
	// ErrBusy returned when reconciliation for the same central is already in progress
	ErrBusy = errors.New("reconciler still busy")
	// ErrCentralNotChanged is an error returned when reconciliation runs more than once in a row with equal central
	ErrCentralNotChanged = errors.New("central not changed")
	// ErrDeletionInProgress returned when central resources are currently deleting
	ErrDeletionInProgress = errors.New("deletion in progress")
	// ErrStopped is returned when the reconciler tried to be started but was already stopped.
	ErrStopped = errors.New("reconciler is stopped")
)

// IsSkippable indicates that the reconciliation was skipped and the status should NOT be reported.
func IsSkippable(err error) bool {
	return errors.Is(err, ErrBusy) ||
		errors.Is(err, ErrCentralNotChanged) ||
		errors.Is(err, ErrDeletionInProgress)
}
