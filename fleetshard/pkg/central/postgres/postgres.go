// Package postgres provides utility functions related to PostreSQL
package postgres

import (
	"fmt"
)

// DBConnection stores the data necessary to connect to a PostgreSQL server
type DBConnection struct {
	Host     string
	Port     int
	User     string
	Database string
}

// CreatePostgresConnectionString generates a connection string from a DBConnection struct
func CreatePostgresConnectionString(connection DBConnection) (string, error) {
	if connection.Host == "" {
		return "", fmt.Errorf("host parameter cannot be empty")
	}
	if connection.Port == 0 {
		return "", fmt.Errorf("port parameter cannot be 0")
	}
	if connection.User == "" {
		return "", fmt.Errorf("user parameter cannot be empty")
	}
	if connection.Database == "" {
		return "", fmt.Errorf("database parameter cannot be empty")
	}

	return fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=require",
		connection.Host, connection.Port, connection.User, connection.Database), nil
}
