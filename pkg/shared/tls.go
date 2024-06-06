package shared

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// TLSWithAdditionalCAs returns a tls config with addiotional trusted ca certificates.
// It uses the systems default certificates and appends the CA certificates in the given files.
func TLSWithAdditionalCAs(caFiles ...string) (*tls.Config, error) {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("failed to load system cert pool: %w", err)
	}

	for _, caFile := range caFiles {
		ca, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read ca file '%s': %w", caFile, err)
		}
		rootCAs.AppendCertsFromPEM(ca)
	}

	return &tls.Config{
		RootCAs: rootCAs,
	}, nil
}
