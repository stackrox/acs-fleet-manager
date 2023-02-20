// Package postgres provides utility functions related to PostreSQL
package postgres

import (
	"fmt"
	"os"
	"sync"
)

// DBConnection stores the data necessary to connect to a PostgreSQL server
type DBConnection struct {
	host     string
	port     int
	database string
	user     string
	password string
}

var (
	once               sync.Once
	rdsCertificateData []byte
)

const (
	// CentralRDSCACertificateBaseName is the name of the additional CA that is passed to Central
	CentralRDSCACertificateBaseName = "rds-ca-bundle"

	sslMode = "verify-full"
	caPath  = "/usr/local/share/ca-certificates/"
	// rdsCACertificatePath stores the location where the RDS CA bundle is mounted in the fleetshard image
	rdsCACertificatePath = caPath + "aws-rds-ca-global-bundle.pem"
	// rdsCACertificatePathCentral stores the location where the RDS CA bundle is mounted in the Central image
	rdsCACertificatePathCentral = caPath + "00-" + CentralRDSCACertificateBaseName + ".crt"
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

// AsConnectionStringForCentral returns a string that can be used by Central to connect to the PostgreSQL server
func (c DBConnection) AsConnectionStringForCentral() string {
	return fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s sslrootcert=%s",
		c.host, c.port, c.user, c.database, sslMode, rdsCACertificatePathCentral)
}

// asConnectionStringForFleetshard returns a string that can be used by fleetshard to connect to a PostgreSQL server. This function
// exposes the password in plain-text, so its output should be used with care.
func (c DBConnection) asConnectionStringForFleetshard() string {
	return fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s sslrootcert=%s password=%s",
		c.host, c.port, c.user, c.database, sslMode, rdsCACertificatePath, c.password)
}

// GetConnectionForUser returns a DBConnection struct for the user given as parameter
func (c DBConnection) GetConnectionForUser(userName string) DBConnection {
	nonPrivilegedConnection := c
	nonPrivilegedConnection.user = userName

	return nonPrivilegedConnection
}

// GetRDSCACertificate returns the location where the RDS CA bundle is mounted in the fleetshard image
func GetRDSCACertificate() ([]byte, error) {
	var err error
	once.Do(func() {
		rdsCertificateData, err = os.ReadFile(rdsCACertificatePath)
	})

	if err != nil {
		return nil, fmt.Errorf("reading RDS CA file: %w", err)
	}

	return rdsCertificateData, nil
}
