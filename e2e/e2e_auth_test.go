package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso"
	"github.com/stackrox/rox/pkg/retry"
)

const (
	internalAPI = "internal"
	publicAPI   = "public"
	adminAPI    = "admin"
)

var _ = Describe("AuthN/Z Fleet* components", func() {

	BeforeEach(func() {
		if env := getEnvDefault("RUN_AUTH_E2E", "false"); env == "false" {
			Skip("The RUN_AUTH_E2E variable was not set, skipping the tests. If you want to run the auth tests, " +
				"set RUN_AUTH_E2E=true")
		}
	})

	fleetManagerEndpoint := "http://localhost:8000"
	if fmEndpointEnv := os.Getenv("FLEET_MANAGER_ENDPOINT"); fmEndpointEnv != "" {
		fleetManagerEndpoint = fmEndpointEnv
	}
	clusterID := "cluster-id"
	if clusterIDEnv := os.Getenv("CLUSTER_ID"); clusterIDEnv != "" {
		clusterID = clusterIDEnv
	}

	env := getEnvDefault("OCM_ENV", "DEVELOPMENT")

	skipOnProd := env == "production"
	skipOnNonProd := env != "production"

	authOption := fleetmanager.OptionFromEnv()

	var client *fleetmanager.Client

	// Needs to be an inline function to allow access to client - passing as arg does not work.
	var testCase = func(group string, fail bool, code int, skip bool) {
		if skip {
			Skip(fmt.Sprintf("Skip test for group %q", group))
		}

		var err error
		switch group {
		case publicAPI:
			_, _, err = client.PublicAPI().GetCentrals(context.Background(), nil)
		case internalAPI:
			_, _, err = client.PrivateAPI().GetCentrals(context.Background(), clusterID)
		case adminAPI:
			_, _, err = client.AdminAPI().GetCentrals(context.Background(), nil)
		default:
			Fail(fmt.Sprintf("Unexpected API Group: %q", group))
		}

		if !fail {
			Expect(err).ToNot(HaveOccurred())
		} else {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(strconv.Itoa(code)))
		}
	}

	Describe("OCM auth type", func() {
		BeforeEach(func() {
			auth, err := fleetmanager.NewOCMAuth(authOption.Ocm)
			Expect(err).ToNot(HaveOccurred())
			fmClient, err := fleetmanager.NewClient(fleetManagerEndpoint, auth)
			Expect(err).ToNot(HaveOccurred())
			client = fmClient
		})

		DescribeTable("AuthN/Z tests",
			testCase,
			Entry("should allow access to fleet manager's public API endpoints",
				publicAPI, false, 0, false),
			Entry("should allow access to fleet manager's internal API endpoints in non-prod environment",
				internalAPI, false, 0, skipOnProd),
			Entry("should not allow access to fleet manager's internal API endpoints in prod environment",
				internalAPI, true, http.StatusNotFound, skipOnNonProd),
			Entry("should not allow access to fleet manager's the admin API",
				adminAPI, true, http.StatusNotFound, false),
		)
	})

	Describe("Static token auth type", func() {
		BeforeEach(func() {
			auth, err := fleetmanager.NewStaticAuth(authOption.Static)
			Expect(err).ToNot(HaveOccurred())
			fmClient, err := fleetmanager.NewClient(fleetManagerEndpoint, auth)
			Expect(err).ToNot(HaveOccurred())
			client = fmClient
		})

		DescribeTable("AuthN/Z tests",
			testCase,
			Entry("should allow access to fleet manager's public API endpoints",
				publicAPI, false, 0, false),
			Entry("should allow access to fleet manager's internal API endpoints in non-prod environment",
				internalAPI, false, 0, skipOnProd),
			Entry("should not allow access to fleet manager's internal API endpoints in prod environment",
				internalAPI, true, http.StatusNotFound, skipOnNonProd),
			Entry("should not allow access to fleet manager's the admin API",
				adminAPI, true, http.StatusNotFound, false),
		)
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

			// Obtain a token from RH SSO using the client ID / secret + client_credentials grant. Write the token to
			// the temporary file.
			token, err := obtainRHSSOToken(clientID, clientSecret)
			Expect(err).ToNot(HaveOccurred())
			_, err = f.WriteString(token)
			Expect(err).ToNot(HaveOccurred())

			// Create the auth type for RH SSO.
			auth, err := fleetmanager.NewRHSSOAuth(fleetmanager.RHSSOOption{TokenFile: f.Name()})
			Expect(err).ToNot(HaveOccurred())
			fmClient, err := fleetmanager.NewClient(fleetManagerEndpoint, auth)
			Expect(err).ToNot(HaveOccurred())
			client = fmClient

			DeferCleanup(func() {
				// Close and delete the temporarily created file.
				err = f.Close()
				Expect(err).ToNot(HaveOccurred())
				err = os.Remove(f.Name())
				Expect(err).ToNot(HaveOccurred())
			})
		})

		DescribeTable("AuthN/Z tests",
			testCase,
			Entry("should allow access to fleet manager's public API endpoints",
				publicAPI, false, 0, false),
			Entry("should allow access to fleet manager's internal API endpoints",
				internalAPI, false, 0, false),
			Entry("should not allow access to fleet manager's the admin API",
				adminAPI, true, http.StatusNotFound, false),
		)
	})
})

// Helpers.

// obtainRHSSOToken will create a redhatsso.SSOClient and retrieve an access token for the specified client ID / secret
// using the client_credentials grant.
func obtainRHSSOToken(clientID, clientSecret string) (string, error) {
	client := redhatsso.NewSSOClient(&iam.IAMConfig{}, &iam.IAMRealmConfig{
		BaseURL:          "https://sso.redhat.com",
		Realm:            "redhat-external",
		ClientID:         clientID,
		ClientSecret:     clientSecret, // pragma: allowlist secret
		TokenEndpointURI: "https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/token",
		JwksEndpointURI:  "https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/certs",
		APIEndpointURI:   "/auth/realms/redhat-external",
	})

	var accessToken string
	err := retry.WithRetry(
		func() error {
			var getTokenErr, retryableErr error
			accessToken, getTokenErr = client.GetToken()
			// Make every error retryable, irrespective of whether the code is transient or not (this is only for test
			// purposes). Ideally, the client itself should handle retries.
			// If we do not check for non-nil errors, MakeRetryable would panic.
			if getTokenErr != nil {
				retryableErr = retry.MakeRetryable(getTokenErr)
			}
			return retryableErr
		},
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttemptNumber int) {
			time.Sleep(10 * time.Second)
		}),
	)
	return accessToken, err
}
