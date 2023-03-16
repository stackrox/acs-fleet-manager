package postgres

import (
	"fmt"
	"os"
)

const (
	// CentralDatabaseCACertificateBaseName is the name of the additional CA that is passed to Central
	CentralDatabaseCACertificateBaseName = "rds-ca-bundle"
	caPath                               = "/usr/local/share/ca-certificates/"
	// DatabaseCACertificatePathFleetshard stores the location where the RDS CA bundle is mounted in the fleetshard image
	DatabaseCACertificatePathFleetshard = caPath + "aws-rds-ca-global-bundle.pem"
	// DatabaseCACertificatePathCentral stores the location where the RDS CA bundle is mounted in the Central image
	DatabaseCACertificatePathCentral = caPath + "00-" + CentralDatabaseCACertificateBaseName + ".crt"
)

// GetDatabaseCACertificates loads the DB server CA certificates from the filesystem, and returns them as a string
// Because the certificates are bundled with the fleetshard-sync image, they are loaded only once.
func GetDatabaseCACertificates() ([]byte, error) {
	var err error
	once.Do(func() {
		rdsCertificateData, err = os.ReadFile(DatabaseCACertificatePathFleetshard)
	})

	if err != nil {
		return nil, fmt.Errorf("reading DB server CA file: %w", err)
	}

	return rdsCertificateData, nil
}
