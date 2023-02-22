package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresConnectionString(t *testing.T) {
	dbConnection, err := NewDBConnection("localhost", 14543, "test-user", "postgresdb")
	require.NoError(t, err)

	require.Equal(t, "host=localhost port=14543 user=test-user dbname=postgresdb sslmode=verify-full", dbConnection.AsConnectionString())

	dbConnectionWithPassword := dbConnection.WithPassword("test_pass")
	require.Equal(t, "host=localhost port=14543 user=test-user dbname=postgresdb sslmode=verify-full",
		dbConnectionWithPassword.AsConnectionString())
	require.Equal(t, "host=localhost port=14543 user=test-user dbname=postgresdb sslmode=verify-full password=test_pass", // pragma: allowlist secret
		dbConnectionWithPassword.asConnectionStringWithPassword())

	dbConnectionWithSSLRootCert := dbConnectionWithPassword.WithSSLRootCert("/tmp/ssl-root-cert.pem")
	require.Equal(t, "host=localhost port=14543 user=test-user dbname=postgresdb sslmode=verify-full sslrootcert=/tmp/ssl-root-cert.pem",
		dbConnectionWithSSLRootCert.AsConnectionString())
	require.Equal(t, "host=localhost port=14543 user=test-user dbname=postgresdb sslmode=verify-full sslrootcert=/tmp/ssl-root-cert.pem password=test_pass", // pragma: allowlist secret
		dbConnectionWithSSLRootCert.asConnectionStringWithPassword())
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
