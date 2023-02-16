package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresConnectionString(t *testing.T) {
	dbConnection, err := NewDBConnection("localhost", 14543, "test-user", "postgresdb")
	require.NoError(t, err)

	require.Equal(t, dbConnection.AsConnectionStringForCentral(),
		"host=localhost port=14543 user=test-user dbname=postgresdb sslmode=verify-full sslrootcert=/usr/local/share/ca-certificates/00-rds-ca-bundle.crt")

	dbConnectionWithPassword := dbConnection.WithPassword("test_pass")
	require.Equal(t, dbConnectionWithPassword.AsConnectionStringForCentral(),
		"host=localhost port=14543 user=test-user dbname=postgresdb sslmode=verify-full sslrootcert=/usr/local/share/ca-certificates/00-rds-ca-bundle.crt")
	require.Equal(t, dbConnectionWithPassword.asConnectionForFleetshard(),
		"host=localhost port=14543 user=test-user dbname=postgresdb sslmode=verify-full sslrootcert=/usr/local/share/ca-certificates/aws-rds-ca-global-bundle.pem password=test_pass") // pragma: allowlist secret
}

func TestNewDBConnection(t *testing.T) {
	_, err := NewDBConnection("", 14543, "test-user", "postgresdb")
	assert.EqualErrorf(t, err, "host parameter cannot be empty", "incorrect error message")

	_, err = NewDBConnection("localhost", 0, "test-user", "postgresdb")
	assert.EqualErrorf(t, err, "port parameter cannot be 0"+
		"", "incorrect error message")

	_, err = NewDBConnection("localhost", 14543, "", "postgresdb")
	assert.EqualErrorf(t, err, "user parameter cannot be empty", "incorrect error message")

	_, err = NewDBConnection("localhost", 14543, "test-user", "")
	assert.EqualErrorf(t, err, "database parameter cannot be empty", "incorrect error message")
}
