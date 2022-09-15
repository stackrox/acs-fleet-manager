package integration

import (
	"net/http"
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/test"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/server"

	. "github.com/onsi/gomega"
	coreTest "github.com/stackrox/acs-fleet-manager/test"
	"github.com/stackrox/acs-fleet-manager/test/mocks"
)

// This tests file ensures that the terms acceptance endpoint is working
const mockDinosaurClusterName = "my-cluster"

type TestEnv struct {
	helper   *coreTest.Helper
	client   *public.APIClient
	teardown func()
}

func termsRequiredSetup(termsRequired bool, t *testing.T) TestEnv {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	termsReviewResponse, err := mocks.GetMockTermsReviewBuilder(nil).TermsRequired(termsRequired).Build()
	if err != nil {
		t.Fatalf(err.Error())
	}
	ocmServerBuilder.SetTermsReviewPostResponse(termsReviewResponse, nil)
	ocmServer := ocmServerBuilder.Build()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, tearDown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(serverConfig *server.ServerConfig, c *config.DataplaneClusterConfig) {
		c.ClusterConfig = config.NewClusterConfig([]config.ManualCluster{test.NewMockDataplaneCluster(mockDinosaurClusterName, 2)})
		serverConfig.EnableTermsAcceptance = true
	})

	return TestEnv{
		helper: h,
		client: client,
		teardown: func() {
			ocmServer.Close()
			tearDown()
		},
	}
}

func TestTermsRequired_CreateDinosaurTermsRequired(t *testing.T) {
	// TODO: Add back this test
	skipNotFullyImplementedYet(t)
	env := termsRequiredSetup(true, t)
	defer env.teardown()

	if test.TestServices.OCMConfig.MockMode != ocm.MockModeEmulateServer {
		t.SkipNow()
	}

	// setup pre-requisites to performing requests
	account := env.helper.NewRandAccount()
	ctx := env.helper.NewAuthenticatedContext(account, nil)

	k := public.CentralRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockDinosaurName,
		MultiAz:       testMultiAZ,
	}

	_, resp, err := env.client.DefaultApi.CreateCentral(ctx, true, k)

	Expect(err).To(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
}

func TestTermsRequired_CreateDinosaur_TermsNotRequired(t *testing.T) {
	// TODO: Add back this test
	skipNotFullyImplementedYet(t)
	env := termsRequiredSetup(false, t)
	defer env.teardown()

	if test.TestServices.OCMConfig.MockMode != ocm.MockModeEmulateServer {
		t.SkipNow()
	}

	// setup pre-requisites to performing requests
	account := env.helper.NewRandAccount()
	ctx := env.helper.NewAuthenticatedContext(account, nil)

	k := public.CentralRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockDinosaurName,
		MultiAz:       testMultiAZ,
	}

	_, resp, err := env.client.DefaultApi.CreateCentral(ctx, true, k)

	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
}
