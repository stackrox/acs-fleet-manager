package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang/glog"
	"github.com/lib/pq"
)

// CentralDBName is the name of database that Central uses. Any name would be acceptable, and the value is
// "central_active" because existing Centrals use it (the name was required to be this one before ACS v4.2.0)
const CentralDBName = "central_active"

// CentralDBInitFunc is a type for functions that perform initialization on a fresh Central DB.
// It requires a valid DBConnection of a user with administrative privileges, and the user name and password
// for a non-privileged user that will be created.
type CentralDBInitFunc func(ctx context.Context, con DBConnection, userName, userPassword string) error

// InitializeDatabase intializes an empty database for a Central:
// - creates a user for Central, with appropriate privileges
// - creates the central_active DB and installs extensions
func InitializeDatabase(ctx context.Context, con DBConnection, userName, userPassword string) error {
	db, err := sql.Open("postgres", con.asConnectionStringWithPassword())
	if err != nil {
		return fmt.Errorf("opening DB: %w", err)
	}

	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			glog.Errorf("Error closing DB: %v", closeErr)
		}
	}()

	err = initializeCentralDBUser(ctx, db, userName, userPassword)
	if err != nil {
		return err
	}

	// We have to create the central_active database here, in order to install extensions.
	// Central won't be able to do it, due to having a limited privileges user.
	err = createCentralDB(ctx, db, CentralDBName, userName, con.user)
	if err != nil {
		return err
	}

	con.database = CentralDBName // extensions are installed in the newly created DB
	err = installExtensions(ctx, con)
	if err != nil {
		return err
	}

	return nil
}

func initializeCentralDBUser(ctx context.Context, db *sql.DB, userName, userPassword string) error {
	err := createNonPrivilegedUser(ctx, db, userName, userPassword)
	if err == nil {
		return nil
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		// It's possible that the Central user already exists, but for some reason (e.g. termination) its
		// password was not stored. In that case, we can simply change its password.
		if pqErr.Code.Name() == "duplicate_object" {
			return changeUserPassword(ctx, db, userName, userPassword)
		}
	}

	return err
}

func createNonPrivilegedUser(ctx context.Context, db *sql.DB, userName, userPassword string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning PostgreSQL transaction: %w", err)
	}

	defer func() {
		switch err {
		case nil:
			err = tx.Commit()
		default:
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				// Just logging the rollback error, because we're already returning the error from the function body
				glog.Errorf("Error rolling back transaction: %v", rollbackErr)
			}
		}
	}()

	// Central needs to be able to create databases
	_, err = tx.ExecContext(ctx, "CREATE USER "+userName+
		" WITH NOSUPERUSER INHERIT CREATEDB NOREPLICATION VALID UNTIL 'infinity' PASSWORD '"+userPassword+"'")
	if err != nil {
		return fmt.Errorf("creating DB user: %w", err)
	}

	// Central needs to use pg_terminate_backend and pg_cancel_backend
	_, err = tx.ExecContext(ctx, "GRANT pg_signal_backend TO "+userName)
	if err != nil {
		return fmt.Errorf("granting pg_signal_backend to Central DB user: %w", err)
	}

	// Central needs access to pg_stat_activity
	_, err = tx.ExecContext(ctx, "GRANT pg_read_all_stats TO "+userName)
	if err != nil {
		return fmt.Errorf("granting pg_read_all_stats to Central DB user: %w", err)
	}

	return nil
}

func changeUserPassword(ctx context.Context, db *sql.DB, userName, userPassword string) error {
	_, err := db.ExecContext(ctx, "ALTER USER "+userName+" WITH PASSWORD '"+userPassword+"'")
	if err != nil {
		return fmt.Errorf("changing the password of DB user %s: %w", userName, err)
	}

	return nil
}

func createCentralDB(ctx context.Context, db *sql.DB, databaseName, owner, currentUser string) error {
	// needed to create a table owned by 'userName'
	_, err := db.ExecContext(ctx, "GRANT "+owner+" TO "+currentUser)
	if err != nil {
		return fmt.Errorf("granting %s role to %s: %w", owner, currentUser, err)
	}

	var queryResult string
	err = db.QueryRowContext(ctx, "SELECT datname FROM pg_database WHERE datname=$1", databaseName).Scan(&queryResult)
	if err != nil {
		if err == sql.ErrNoRows {
			_, err = db.ExecContext(ctx, "CREATE DATABASE "+databaseName+" OWNER "+owner)
			if err != nil {
				return fmt.Errorf("creating central_active DB: %w", err)
			}
		} else {
			return fmt.Errorf("checking if central_active DB exists: %w", err)
		}
	}

	return nil
}

func installExtensions(ctx context.Context, con DBConnection) error {
	db, err := sql.Open("postgres", con.asConnectionStringWithPassword())
	if err != nil {
		return fmt.Errorf("opening DB: %w", err)
	}

	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			glog.Errorf("Error closing DB: %v", closeErr)
		}
	}()

	return installExtensionsOnCentralDB(ctx, db)
}

func installExtensionsOnCentralDB(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS pg_stat_statements")
	if err != nil {
		return fmt.Errorf("creating extension: %w", err)
	}

	return nil
}
