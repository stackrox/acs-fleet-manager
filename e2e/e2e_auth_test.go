package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	fmImpl "github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager/impl"
)

const (
	internalAPI = "internal"
	publicAPI   = "public"
	adminAPI    = "admin"
)

var _ = Describe("AuthN/Z Fleet* components", Ordered, func() {

	BeforeEach(func() {
		if !runAuthTests {
			Skip("Skipping auth test")
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

	authOption := fmImpl.OptionFromEnv()

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
			auth, err := fmImpl.NewOCMAuth(context.Background(), authOption.Ocm)
			Expect(err).ToNot(HaveOccurred())
			fmClient, err := fmImpl.NewClient(fleetManagerEndpoint, auth)
			Expect(err).ToNot(HaveOccurred())
			client = fmClient
		})

		DescribeTable("AuthN/Z tests",
			testCase,
			Entry("should allow access to fleet manager's public API endpoints",
				publicAPI, false, 0, false),
			Entry("should not allow access to fleet manager's internal API endpoints",
				internalAPI, true, http.StatusNotFound, false),
			Entry("should not allow access to fleet manager's admin API endpoints",
				adminAPI, true, http.StatusNotFound, false),
		)
	})

	Describe("Static token auth type", func() {
		BeforeEach(func() {
			auth, err := fmImpl.NewStaticAuth(context.Background(), authOption.Static)
			Expect(err).ToNot(HaveOccurred())
			fmClient, err := fmImpl.NewClient(fleetManagerEndpoint, auth)
			Expect(err).ToNot(HaveOccurred())
			client = fmClient
		})

		DescribeTable("AuthN/Z tests",
			testCase,
			Entry("should allow access to fleet manager's public API endpoints",
				publicAPI, false, 0, false),
			Entry("should not allow access to fleet manager's internal API endpoints",
				internalAPI, true, http.StatusUnauthorized, false),
			Entry("should not allow access to fleet manager's admin API endpoints",
				adminAPI, true, http.StatusNotFound, false),
		)
	})
	// TEST ME
	Describe("RH SSO auth type", func() {
		BeforeEach(func() {
			rhSSOOpt := authOption.Sso
			// Skip the tests if client ID / secret read from environment variables not set.
			if rhSSOOpt.ClientID == "" || rhSSOOpt.ClientSecret == "" {
				Skip("RHSSO_SERVICE_ACCOUNT_CLIENT_ID / RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET not set, cannot initialize auth type")
			}
			// Create the auth type for RH SSO.
			auth, err := fmImpl.NewRHSSOAuth(context.Background(), rhSSOOpt)
			Expect(err).ToNot(HaveOccurred())
			fmClient, err := fmImpl.NewClient(fleetManagerEndpoint, auth)
			Expect(err).ToNot(HaveOccurred())
			client = fmClient
		})

		DescribeTable("AuthN/Z tests",
			testCase,
			Entry("should allow access to fleet manager's public API endpoints",
				publicAPI, false, 0, false),
			Entry("should not allow access to fleet manager's internal API endpoints",
				internalAPI, true, http.StatusNotFound, false),
			Entry("should not allow access to fleet manager's admin API endpoints",
				adminAPI, true, http.StatusNotFound, false),
		)
	})
})
