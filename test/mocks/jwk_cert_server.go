package mocks

import (
	"crypto"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mendsley/gojwk"
)

const (
	certEndpoint = "/auth/realms/rhd/protocol/openid-connect/certs"
	rhs256       = "RHS256"
)

// NewJWKCertServerMock ...
func NewJWKCertServerMock(t *testing.T, pubKey crypto.PublicKey, jwkKID string) (url string, teardown func()) {
	certHandler := http.NewServeMux()
	certHandler.HandleFunc(certEndpoint,
		func(w http.ResponseWriter, r *http.Request) {
			pubjwk, err := gojwk.PublicKey(pubKey)
			if err != nil {
				t.Errorf("Unable to generate public jwk: %s", err)
				return
			}
			pubjwk.Kid = jwkKID
			pubjwk.Alg = rhs256
			jwkBytes, err := gojwk.Marshal(pubjwk)
			if err != nil {
				t.Errorf("Unable to marshal public jwk: %s", err)
				return
			}
			fmt.Fprintf(w, fmt.Sprintf(`{"keys":[%s]}`, string(jwkBytes))) //nolint:staticcheck
		},
	)

	server := httptest.NewServer(certHandler)
	return fmt.Sprintf("%s%s", server.URL, certEndpoint), server.Close
}
