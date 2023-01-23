package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

const centralDBName = "central_active"

// CentralDBInitFunc is a type for functions that perform initialization on a fresh Central DB.
// It requires a valid DBConnection of a user with administrative privileges, and the user name and password
// for a non-privileged user that will be created.
type CentralDBInitFunc func(ctx context.Context, con DBConnection, userName, userPassword string) error

// InitializeDatabase intializes an empty database for a Central:
// - creates a user for Central, with appropriate privileges
// - creates the central_active DB and installs extensions
func InitializeDatabase(ctx context.Context, con DBConnection, userName, userPassword string) error {
	db, err := sql.Open("postgres", con.AsConnectionStringWithPassword())
	if err != nil {
		return fmt.Errorf("opening DB: %w", err)
	}

	defer db.Close()

	err = initializeNonPrivilegedUser(ctx, db, userName, userPassword)
	if err != nil {
		return err
	}

	// We have to create the central_active database here, in order to install extensions.
	// Central won't be able to do it, due to having a limited privileges user.
	err = createCentralDB(ctx, db, centralDBName, userName, con.user)
	if err != nil {
		return err
	}

	con.database = centralDBName // extensions are installed in the newly created DB
	err = installExtensions(ctx, con)
	if err != nil {
		return err
	}

	return nil
}

func initializeNonPrivilegedUser(ctx context.Context, db *sql.DB, userName, userPassword string) error {
	err := createUser(ctx, db, userName, userPassword)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			// It's possible that the non privileged user already exists, but for some reason (e.g. termination) its
			// password was not stored. In that case, we can simply change its password.
			if pqErr.Code.Name() == "duplicate_object" {
				return changeUserPassword(ctx, db, userName, userPassword)
			}
		}

		return err
	}

	return nil
}

func createUser(ctx context.Context, db *sql.DB, userName, userPassword string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("beginning PostgreSQL transaction: %w", err)
	}

	defer func() {
		switch err {
		case nil:
			err = tx.Commit()
		default:
			tx.Rollback()
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
	db, err := sql.Open("postgres", con.AsConnectionStringWithPassword())
	if err != nil {
		return fmt.Errorf("opening DB: %w", err)
	}

	defer db.Close()

	_, err = db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS pg_stat_statements")
	if err != nil {
		return fmt.Errorf("creating extension: %w", err)
	}

	return nil
}
