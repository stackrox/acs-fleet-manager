package test

import (
	"fmt"
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
	"github.com/stackrox/acs-fleet-manager/test"
)

func NewApiClient(helper *test.Helper) *public.APIClient {
	var serverConfig *server.ServerConfig
	helper.Env.MustResolveAll(&serverConfig)

	openapiConfig := public.NewConfiguration()
	openapiConfig.BasePath = fmt.Sprintf("http://%s", serverConfig.BindAddress)
	client := public.NewAPIClient(openapiConfig)
	return client
}
