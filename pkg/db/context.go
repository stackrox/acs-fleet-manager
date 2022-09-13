package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	logger "github.com/stackrox/acs-fleet-manager/pkg/logging"
)

type contextKey int

const (
	transactionKey contextKey = iota
)

// NewContext returns a new context with transaction stored in it.
// Upon error, the original context is still returned along with an error
func (c *ConnectionFactory) NewContext(ctx context.Context) (context.Context, error) {
	tx, err := c.newTransaction()
	if err != nil {
		return ctx, err
	}

	// adding txid explicitly to context with a simple string key and int value
	// due to a cyclical import cycle between pkg/db and pkg/logging
	ctx = context.WithValue(ctx, "txid", tx.txid) //nolint:staticcheck,revive
	ctx = context.WithValue(ctx, transactionKey, tx)

	return ctx, nil
}

// TxContext creates a new transaction context from context.Background()
func (c *ConnectionFactory) TxContext() (ctx context.Context, err error) {
	return c.NewContext(context.Background())
}

// Resolve resolves the current transaction according to the rollback flag.
func Resolve(ctx context.Context) error {
	tx, ok := ctx.Value(transactionKey).(*txFactory)
	if !ok {
		return fmt.Errorf("Could not retrieve transaction from context")
	}
	if tx.resolved {
		return nil
	}
	tx.resolved = true
	postCommitActions := tx.postCommitActions
	tx.postCommitActions = nil
	if tx.markedForRollback() {
		if err := tx.tx.Rollback(); err != nil {
			return fmt.Errorf("Could not rollback transaction: %v", err)
		}
		ulog := logger.NewUHCLogger(ctx)
		ulog.Infof("Rolled back transaction")
	} else {
		if err := tx.tx.Commit(); err != nil {
			// TODO:  what does the user see when this occurs? seems like they will get a false positive
			return fmt.Errorf("Could not commit transaction: %v", err)
		}
		for _, f := range postCommitActions {
			f()
		}
	}
	return nil
}

// Begin ...
func Begin(ctx context.Context) error {
	tx, ok := ctx.Value(transactionKey).(*txFactory)
	if !ok {
		return fmt.Errorf("Could not retrieve transaction from context")
	}

	err := tx.begin()
	if err != nil {
		return fmt.Errorf("Could not begin transaction: %v", err)
	}
	return nil
}

// AddPostCommitAction ...
func AddPostCommitAction(ctx context.Context, f func()) error {
	tx, ok := ctx.Value(transactionKey).(*txFactory)
	if !ok {
		return fmt.Errorf("Could not retrieve transaction from context")
	}

	tx.postCommitActions = append(tx.postCommitActions, f)
	return nil
}

// FromContext Retrieves the transaction from the context.
func FromContext(ctx context.Context) (*sql.Tx, error) {
	transaction, ok := ctx.Value(transactionKey).(*txFactory)
	if !ok {
		return nil, errors.GeneralError("Could not retrieve transaction from context")
	}
	return transaction.tx, nil
}

// MarkForRollback flags the transaction stored in the context for rollback and logs whatever error caused the rollback
func MarkForRollback(ctx context.Context, err error) {
	ulog := logger.NewUHCLogger(ctx)
	transaction, ok := ctx.Value(transactionKey).(*txFactory)
	if !ok {
		ulog.Errorf("failed to mark transaction for rollback: could not retrieve transaction from context")
		return
	}
	ulog.Infof("Marked transaction for rollback, err: %v", err)
	transaction.rollbackFlag = true
}
