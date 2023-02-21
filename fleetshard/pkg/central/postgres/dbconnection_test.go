package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresConnectionString(t *testing.T) {
	dbConnection, err := NewDBConnection("localhost", 14543, "test-user", "postgresdb")
	require.NoError(t, err)

	require.Equal(t, dbConnection.AsConnectionString(), "host=localhost port=14543 user=test-user dbname=postgresdb sslmode=require")

	dbConnectionWithPassword := dbConnection.WithPassword("test_pass")
	require.Equal(t, dbConnectionWithPassword.AsConnectionString(), "host=localhost port=14543 user=test-user dbname=postgresdb sslmode=require")
	require.Equal(t, dbConnectionWithPassword.asConnectionStringWithPassword(),
		"host=localhost port=14543 user=test-user dbname=postgresdb sslmode=require password=test_pass") // pragma: allowlist secret
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
