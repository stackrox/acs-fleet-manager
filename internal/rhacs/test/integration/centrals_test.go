package integration

import (
	"fmt"
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/rhacs"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
	"github.com/stackrox/acs-fleet-manager/test"
	"github.com/stackrox/acs-fleet-manager/test/mocks"
)

// FIXME move to common place
func NewApiClient(helper *test.Helper) *public.APIClient {
	var serverConfig *server.ServerConfig
	helper.Env.MustResolveAll(&serverConfig)

	openapiConfig := public.NewConfiguration()
	openapiConfig.BasePath = fmt.Sprintf("http://%s", serverConfig.BindAddress)
	client := public.NewAPIClient(openapiConfig)
	return client
}

func TestCentralGet(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	testHelper, teardown := test.NewHelperWithHooks(t, ocmServer, nil, rhacs.ConfigProviders())
	defer teardown()
	client := NewApiClient(testHelper)

	account := testHelper.NewRandAccount()
	ctx := testHelper.NewAuthenticatedContext(account, nil)

	central, response, _ := client.DefaultApi.GetCentralById(ctx, "missingId")
	fmt.Printf("central %v, response %v", central, response)
	fmt.Println("bye for now")
}