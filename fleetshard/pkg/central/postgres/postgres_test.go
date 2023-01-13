package postgres

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPostgresConnectionString(t *testing.T) {
	dbConnection := DBConnection{
		Host:     "localhost",
		Port:     14543,
		User:     "test-user",
		Database: "postgresdb",
	}

	dbConnectionString, err := CreatePostgresConnectionString(dbConnection)
	require.NoError(t, err)
	require.Equal(t, dbConnectionString, "host=localhost port=14543 user=test-user dbname=postgresdb sslmode=require")

	dbConnection.Host = ""
	_, err = CreatePostgresConnectionString(dbConnection)
	require.Error(t, err)

	dbConnection.Host = "host"
	dbConnection.Port = 0
	_, err = CreatePostgresConnectionString(dbConnection)
	require.Error(t, err)

	dbConnection.Port = 5432
	dbConnection.User = ""
	_, err = CreatePostgresConnectionString(dbConnection)
	require.Error(t, err)

	dbConnection.User = "postgres"
	dbConnection.Database = ""
	_, err = CreatePostgresConnectionString(dbConnection)
	require.Error(t, err)
}
