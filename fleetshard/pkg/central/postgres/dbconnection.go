// Package postgres provides utility functions related to PostreSQL
package postgres

import (
	"fmt"
	"sync"
)

// DBConnection stores the data necessary to connect to a PostgreSQL server
type DBConnection struct {
	host        string
	port        int
	database    string
	user        string
	password    string
	sslrootcert string
}

var (
	once               sync.Once
	rdsCertificateData []byte
)

const (
	sslMode          = "verify-full"
	statementTimeout = 1200000
	clientEncoding   = "UTF8"
)

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

// WithSSLRootCert adds an optional sslrootcert parameter to the DBConnection struct, which points PostgreSQL to the
// location of the CA root certificate
func (c DBConnection) WithSSLRootCert(sslrootcert string) DBConnection {
	c.sslrootcert = sslrootcert
	return c
}

// AsConnectionString returns a string that can be used to connect to a PostgreSQL server. The password is omitted.
func (c DBConnection) AsConnectionString() string {
	connectionString := fmt.Sprintf("host=%s port=%d user=%s dbname=%s statement_timeout=%d client_encoding=%s sslmode=%s",
		c.host, c.port, c.user, c.database, statementTimeout, clientEncoding, sslMode)
	if c.sslrootcert != "" {
		connectionString = fmt.Sprintf("%s sslrootcert=%s", connectionString, c.sslrootcert)
	}

	return connectionString
}

// asConnectionStringWithPassword returns a string that can be used to connect to a PostgreSQL server
func (c DBConnection) asConnectionStringWithPassword() string {
	return c.AsConnectionString() + fmt.Sprintf(" password=%s", c.password)
}

// GetConnectionForUser returns a DBConnection struct for the user given as parameter
func (c DBConnection) GetConnectionForUserAndDB(userName, dbName string) DBConnection {
	newConnection := c
	newConnection.user = userName
	newConnection.database = dbName

	return newConnection
}
