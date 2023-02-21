// Package postgres provides utility functions related to PostreSQL
package postgres

import (
	"fmt"
)

// DBConnection stores the data necessary to connect to a PostgreSQL server
type DBConnection struct {
	host     string
	port     int
	database string
	user     string
	password string
}

const sslMode = "require"

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
		database: database,
		user:     user,
	}, nil
}

// WithPassword adds an optional password to the DBConnection struct
func (c DBConnection) WithPassword(password string) DBConnection {
	c.password = password // pragma: allowlist secret
	return c
}

// AsConnectionString returns a string that can be used to connect to a PostgreSQL server. The password is omitted.
func (c DBConnection) AsConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
		c.host, c.port, c.user, c.database, sslMode)
}

// asConnectionStringWithPassword returns a string that can be used to connect to a PostgreSQL server. This function
// exposes the password in plain-text, so it should be used with care.
func (c DBConnection) asConnectionStringWithPassword() string {
	return c.AsConnectionString() + fmt.Sprintf(" password=%s", c.password)
}

// GetConnectionForUser returns a DBConnection struct for the user given as parameter
func (c DBConnection) GetConnectionForUser(userName string) DBConnection {
	nonPrivilegedConnection := c
	nonPrivilegedConnection.user = userName

	return nonPrivilegedConnection
}
