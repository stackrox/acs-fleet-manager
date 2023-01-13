// Package postgres provides utility functions related to PostreSQL
package postgres

import (
	"fmt"
)

// DBConnection stores the data necessary to connect to a PostgreSQL server
type DBConnection struct {
	host     string
	port     int
	user     string
	database string
}

// NewDBConnection constructs a new DBConnection struct
func NewDBConnection(host string, port int, user, database string) (DBConnection, error) {
	if host == "" {
		return DBConnection{}, fmt.Errorf("host parameter cannot be empty")
	}
	if port == 0 {
		return DBConnection{}, fmt.Errorf("port parameter cannot be 0")
	}
	if user == "" {
		return DBConnection{}, fmt.Errorf("user parameter cannot be empty")
	}
	if database == "" {
		return DBConnection{}, fmt.Errorf("database parameter cannot be empty")
	}

	return DBConnection{
		host:     host,
		port:     port,
		user:     user,
		database: database,
	}, nil
}

// AsConnectionString returns a string that can be used to connect to a PostgreSQL server
func (c DBConnection) AsConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=require",
		c.host, c.port, c.user, c.database)
}
