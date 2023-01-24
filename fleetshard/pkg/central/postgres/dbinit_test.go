package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/lib/pq"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestInitializeNonPrivilegedUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("opening a stub database connection: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("CREATE USER").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("GRANT pg_signal_backend").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("GRANT pg_read_all_stats").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = initializeNonPrivilegedUser(context.TODO(), db, "test_user", "user_pass")
	require.NoError(t, err)

	// we make sure that all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestInitializeNonPrivilegedUserError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("opening a stub database connection: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("CREATE USER").WillReturnError(fmt.Errorf("some error"))
	mock.ExpectRollback()

	err = initializeNonPrivilegedUser(context.TODO(), db, "test_user", "user_pass")
	require.Error(t, err)

	// we make sure that all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestInitializeNonPrivilegedUserAlreadyExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("opening a stub database connection: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	var pqErr pq.Error
	pqErr.Code = "42710" // code of "duplicate_object"
	// test that if user already exists, then a password change will be performed instead
	mock.ExpectExec("CREATE USER").WillReturnError(&pqErr)
	mock.ExpectRollback()
	mock.ExpectExec("ALTER USER").WillReturnResult(sqlmock.NewResult(1, 1))

	err = initializeNonPrivilegedUser(context.TODO(), db, "test_user", "user_pass")
	require.NoError(t, err)

	// we make sure that all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestCreateCentralDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("opening a stub database connection: %v", err)
	}
	defer db.Close()

	columns := []string{"datname"}
	mock.ExpectExec("GRANT rhacs_central TO rhacs_master").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT datname FROM pg_database").WithArgs("central_active").WillReturnRows(
		sqlmock.NewRows(columns))
	mock.ExpectExec("CREATE DATABASE central_active OWNER rhacs_central").WillReturnResult(sqlmock.NewResult(1, 1))

	err = createCentralDB(context.TODO(), db, "central_active", "rhacs_central", "rhacs_master")
	require.NoError(t, err)

	// we make sure that all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestCreateCentralDBAlreadyExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("opening a stub database connection: %v", err)
	}
	defer db.Close()

	columns := []string{"datname"}
	mock.ExpectExec("GRANT rhacs_central TO rhacs_master").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT datname FROM pg_database").WithArgs("central_active").WillReturnRows(
		sqlmock.NewRows(columns).AddRow("central_active"))

	err = createCentralDB(context.TODO(), db, "central_active", "rhacs_central", "rhacs_master")
	require.NoError(t, err)

	// we make sure that all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestInstallExtensions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("opening a stub database connection: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE EXTENSION IF NOT EXISTS pg_stat_statements").WillReturnResult(sqlmock.NewResult(1, 1))

	err = installExtensionsOnCentralDB(context.TODO(), db)
	require.NoError(t, err)

	// we make sure that all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}
