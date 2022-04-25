package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	g "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
	"github.com/stackrox/acs-fleet-manager/test"
	"github.com/stackrox/acs-fleet-manager/test/mocks"
)

// Hack as in https://github.com/operator-framework/operator-sdk/issues/2913 and suggested in https://github.com/onsi/ginkgo/issues/9
var Testing *testing.T

func TestCentralResource(t *testing.T) {
	RegisterFailHandler(g.Fail)
	Testing = t
	g.RunSpecs(t, "Central API resource Suite")
}

func NewApiClient(helper *test.Helper) *public.APIClient {
	var serverConfig *server.ServerConfig
	helper.Env.MustResolveAll(&serverConfig)

	openapiConfig := public.NewConfiguration()
	openapiConfig.BasePath = fmt.Sprintf("http://%s", serverConfig.BindAddress)
	client := public.NewAPIClient(openapiConfig)
	return client
}

var _ = g.Describe("Central API resource", func() {
	var (
		t   *testing.T
		ocmServer *httptest.Server
		testHelper *test.Helper
		client *public.APIClient
		teardown func()
		ctx context.Context
	) 
	
	g.BeforeEach(func() {
		t = Testing
		ocmServer = mocks.NewMockConfigurableServerBuilder().Build()
		testHelper, teardown = test.NewHelperWithHooks(t, ocmServer, nil, rhacs.ConfigProviders())

		client = NewApiClient(testHelper)

		account := testHelper.NewRandAccount()
		ctx = testHelper.NewAuthenticatedContext(account, nil)
	})

	g.AfterEach(func() {
		ocmServer.Close()
		teardown()
	})

	g.Context("the database is empty, and an authorized user makes a request", func() {
		g.When("listing all centrals", func() {
			g.It("returns an empty centrals list", func() {
				centrals, response, err := client.DefaultApi.GetCentrals(ctx, nil)
				Expect(err).NotTo(HaveOccurred(), "Error listing centrals: %v", err)
				Expect(response.StatusCode).To(Equal(http.StatusOK))
				Expect(centrals.Kind).To(Equal("CentralRequestList"))
				Expect(centrals.Items).To(BeEmpty())
				Expect(centrals.NextPageCursor).To(BeEmpty())
				Expect(centrals.Size).To(BeEquivalentTo(0))
			})
		})

		g.When("getting a missing central", func() {
			g.It("returns not found error", func() {
				_, response, err := client.DefaultApi.GetCentralById(ctx, "missing ID")
				Expect(err).Error()
				Expect(response.StatusCode).To(Equal(http.StatusNotFound))
			})
			
		})

	})
})
