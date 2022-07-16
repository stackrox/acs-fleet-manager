package e2e

import (
	"encoding/json"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/compat"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso"
	"io"
	"net/http"
	"os"
	"strconv"
)

const (
	ocmAuthType         = "OCM"
	rhSSOAuthType       = "RHSSO"
	staticTokenAuthType = "STATIC_TOKEN"
)

var _ = Describe("AuthN/Z Fleet* components", func() {
	if env := getEnvDefault("RUN_AUTH_E2E", "false"); env == "false" {
		Skip("The RUN_AUTH_E2E variable was not set, skipping the tests. If you want to run the auth tests, " +
			"set RUN_AUTH_E2E=true")
	}

	defer GinkgoRecover()

	var client *authTestClientFleetManager

	Describe("OCM auth type", func() {
		BeforeEach(func() {
			auth, err := fleetmanager.NewAuth(ocmAuthType)
			Expect(err).ToNot(HaveOccurred())
			fmClient, err := fleetmanager.NewClient("http://localhost:8000", "1234567890abcdef1234567890abcdef", auth)
			Expect(err).ToNot(HaveOccurred())
			client = newAuthTestClient(fmClient, auth, "http://localhost:8000")
		})

		It("should allow access to fleet manager's public API endpoints", func() {
			_, err := client.ListCentrals()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should allow access to fleet manager's internal API endpoints in dev / staging environment", func() {
			_, err := client.GetManagedCentralList()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not allow access to fleet manager's internal API endpoints in production environment", func() {
			// OCM_ENV specifies which environment configuration to use when deploying fleet manager.
			if env := getEnvDefault("OCM_ENV", "DEVELOPMENT"); env != "production" {
				Skip("Fleet manager is deployed using non-production configuration settings")
			}

			_, err := client.GetManagedCentralList()
			Expect(err).To(HaveOccurred())
			// Currently the errors exposed by the generated open API clients do not satisfy the error interface, hence
			// we can't check for error types via MatchError matcher but have to resort to string checking.
			// Instead of http.StatusUnauthorized, we will retrieve http.StatusNotFound.
			Expect(err.Error()).To(ContainSubstring(strconv.Itoa(http.StatusNotFound)))
		})

		It("should not allow access to fleet manager's the admin API", func() {
			_, err := client.ListAdminAPI()

			Expect(err).To(HaveOccurred())
			// Currently the errors exposed by the generated open API clients do not satisfy the error interface, hence
			// we can't check for error types via MatchError matcher but have to resort to string checking.
			// Instead of http.StatusUnauthorized, we will retrieve http.StatusNotFound.
			Expect(err.Error()).To(ContainSubstring(strconv.Itoa(http.StatusNotFound)))
		})
	})

	Describe("Static token auth type", func() {
		BeforeEach(func() {
			auth, err := fleetmanager.NewAuth(staticTokenAuthType)
			Expect(err).ToNot(HaveOccurred())
			fmClient, err := fleetmanager.NewClient("http://localhost:8000", "cluster-id", auth)
			Expect(err).ToNot(HaveOccurred())
			client = newAuthTestClient(fmClient, auth, "http://localhost:8000")
		})

		It("should allow access to fleet manager's public API endpoints", func() {
			_, err := client.ListCentrals()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should allow access to fleet manager's internal API endpoints in dev / staging environment", func() {
			_, err := client.GetManagedCentralList()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not allow access to fleet manager's internal API endpoints in production environment", func() {
			// OCM_ENV specifies which environment configuration to use when deploying fleet manager.
			if env := getEnvDefault("OCM_ENV", "DEVELOPMENT"); env != "production" {
				Skip("Fleet manager is deployed using non-production configuration settings")
			}

			_, err := client.GetManagedCentralList()
			Expect(err).To(HaveOccurred())
			// Currently the errors exposed by the generated open API clients do not satisfy the error interface, hence
			// we can't check for error types via MatchError matcher but have to resort to string checking.
			// Instead of http.StatusUnauthorized, we will retrieve http.StatusNotFound.
			Expect(err.Error()).To(ContainSubstring(strconv.Itoa(http.StatusNotFound)))
		})

		It("should not allow access to fleet manager's the admin API", func() {
			_, err := client.ListAdminAPI()

			Expect(err).To(HaveOccurred())
			// Currently the errors exposed by the generated open API clients do not satisfy the error interface, hence
			// we can't check for error types via MatchError matcher but have to resort to string checking.
			// Instead of http.StatusUnauthorized, we will retrieve http.StatusNotFound.
			Expect(err.Error()).To(ContainSubstring(strconv.Itoa(http.StatusNotFound)))
		})
	})

	Describe("RH SSO auth type", func() {
		BeforeEach(func() {
			// Read the client ID / secret from environment variables. If not set, skip the tests.
			clientID := os.Getenv("RHSSO_CLIENT_ID")
			clientSecret := os.Getenv("RHSSO_CLIENT_SECRET")
			if clientID == "" || clientSecret == "" {
				Skip("RHSSO_CLIENT_ID / RHSSO_CLIENT_SECRET not set, cannot initialize auth type")
			}

			// Create a temporary file where the token will be stored.
			f, err := os.CreateTemp("", "token")
			Expect(err).ToNot(HaveOccurred())

			// Set the RHSSO_TOKEN_FILE environment variable, pointing to the temporary file.
			err = os.Setenv("RHSSO_TOKEN_FILE", f.Name())
			Expect(err).ToNot(HaveOccurred())

			// Obtain a token from RH SSO using the client ID / secret + client_credentials grant. Write the token to
			// the temporary file.
			token, err := obtainRHSSOToken(clientID, clientSecret)
			Expect(err).ToNot(HaveOccurred())
			_, err = f.WriteString(token)
			Expect(err).ToNot(HaveOccurred())

			// Create the auth type for RH SSO.
			auth, err := fleetmanager.NewAuth(rhSSOAuthType)
			Expect(err).ToNot(HaveOccurred())
			fmClient, err := fleetmanager.NewClient("http://localhost:8000", "cluster-id", auth)
			Expect(err).ToNot(HaveOccurred())
			client = newAuthTestClient(fmClient, auth, "http://localhost:8000")

			DeferCleanup(func() {
				// Unset the environment variable.
				err := os.Unsetenv("RHSSO_TOKEN_FILE")
				Expect(err).ToNot(HaveOccurred())

				// Close and delete the temporarily created file.
				err = f.Close()
				Expect(err).ToNot(HaveOccurred())
				err = os.Remove(f.Name())
				Expect(err).ToNot(HaveOccurred())
			})
		})

		It("should allow access to fleet manager's public API endpoints", func() {
			_, err := client.ListCentrals()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should allow access to fleet manager's internal API endpoints in dev / staging environment", func() {
			_, err := client.GetManagedCentralList()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not allow access to fleet manager's internal API endpoints in production environment", func() {
			// OCM_ENV specifies which environment configuration to use when deploying fleet manager.
			if env := getEnvDefault("OCM_ENV", "DEVELOPMENT"); env != "production" {
				Skip("Fleet manager is deployed using non-production configuration settings")
			}

			_, err := client.GetManagedCentralList()
			Expect(err).To(HaveOccurred())
			// Currently the errors exposed by the generated open API clients do not satisfy the error interface, hence
			// we can't check for error types via MatchError matcher but have to resort to string checking.
			// Instead of http.StatusUnauthorized, we will retrieve http.StatusNotFound.
			Expect(err.Error()).To(ContainSubstring(strconv.Itoa(http.StatusNotFound)))
		})

		It("should not allow access to fleet manager's the admin API", func() {
			_, err := client.ListAdminAPI()

			Expect(err).To(HaveOccurred())
			// Currently the errors exposed by the generated open API clients do not satisfy the error interface, hence
			// we can't check for error types via MatchError matcher but have to resort to string checking.
			// Instead of http.StatusUnauthorized, we will retrieve http.StatusNotFound.
			Expect(err.Error()).To(ContainSubstring(strconv.Itoa(http.StatusNotFound)))
		})
	})
})

// Helpers.

// authTestClientFleetManager embeds the fleetmanager.Client and adds additional method for admin API (which shouldn't
// be a part of the fleetmanager.Client as it is only used within tests).
type authTestClientFleetManager struct {
	*fleetmanager.Client
	auth     fleetmanager.Auth
	h        http.Client
	endpoint string
}

func newAuthTestClient(c *fleetmanager.Client, auth fleetmanager.Auth, endpoint string) *authTestClientFleetManager {
	return &authTestClientFleetManager{c, auth, http.Client{}, endpoint}
}

func (a *authTestClientFleetManager) ListAdminAPI() (*private.DinosaurList, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", a.endpoint, "admin/dinosaurs"), nil)
	if err != nil {
		return nil, err
	}

	if err := a.auth.AddAuth(req); err != nil {
		return nil, err
	}

	resp, err := a.h.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	into := struct {
		Kind string `json:"kind"`
	}{}
	err = json.Unmarshal(data, &into)
	if err != nil {
		return nil, err
	}

	// Unmarshal error
	if into.Kind == "Error" || into.Kind == "error" {
		apiError := compat.Error{}
		err = json.Unmarshal(data, &apiError)
		if err != nil {
			return nil, err
		}
		return nil, errors.Errorf("API error (HTTP status %d) occured %s: %s", resp.StatusCode, apiError.Code, apiError.Reason)
	}

	var dinosaurList *private.DinosaurList

	err = json.Unmarshal(data, dinosaurList)
	return dinosaurList, err
}

// obtainRHSSOToken will create a redhatsso.SSOClient and retrieve an access token for the specified client ID / secret
// using the client_credentials grant.
func obtainRHSSOToken(clientID, clientSecret string) (string, error) {
	client := redhatsso.NewSSOClient(&iam.IAMConfig{}, &iam.IAMRealmConfig{
		BaseURL:          "https://sso.redhat.com",
		Realm:            "redhat-external",
		ClientID:         clientID,
		ClientSecret:     clientSecret,
		TokenEndpointURI: "https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/token",
		JwksEndpointURI:  "https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/certs",
		APIEndpointURI:   "/auth/realms/redhat-external",
	})
	return client.GetToken()
}
