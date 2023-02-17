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

const sslMode = "verify-full"

const caPath = "/usr/local/share/ca-certificates/"

// RDSCertificatePath stores the location where the RDS CA bundle is mounted in the fleetshard image
const RDSCertificatePath = caPath + "aws-rds-ca-global-bundle.pem"

// CentralRDSCertificateBaseName is the name of the additional CA that is passed to Central
const CentralRDSCertificateBaseName = "rds-ca-bundle"

// rdsCertificatePathCentral stores the location where the RDS CA bundle is mounted in the Central image
const rdsCertificatePathCentral = caPath + "00-" + CentralRDSCertificateBaseName + ".crt"

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

// AsConnectionStringForCentral returns a string that can be used by Central to connect to the PostgreSQL server
func (c DBConnection) AsConnectionStringForCentral() string {
	return fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s sslrootcert=%s",
		c.host, c.port, c.user, c.database, sslMode, rdsCertificatePathCentral)
}

// asConnectionStringForFleetshard returns a string that can be used by fleetshard to connect to a PostgreSQL server. This function
// exposes the password in plain-text, so its output should be used with care.
func (c DBConnection) asConnectionStringForFleetshard() string {
	return fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s sslrootcert=%s password=%s",
		c.host, c.port, c.user, c.database, sslMode, RDSCertificatePath, c.password)
}

// GetConnectionForUser returns a DBConnection struct for the user given as parameter
func (c DBConnection) GetConnectionForUser(userName string) DBConnection {
	nonPrivilegedConnection := c
	nonPrivilegedConnection.user = userName

	return nonPrivilegedConnection
}
