package multicluster

import (
	"context"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	fmImpl "github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager/impl"
)

var _ = Describe("Central", Ordered, func() {
	var fleetmanagerClient *fleetmanager.Client
	var fleetmanagerAdminClient fleetmanager.AdminAPI

	BeforeAll(func() {
		options := fmImpl.OptionFromEnv()
		auth, err := fmImpl.NewStaticAuth(context.Background(), fmImpl.StaticOption{StaticToken: options.Static.StaticToken})
		Expect(err).ToNot(HaveOccurred())
		fleetmanagerClient, err = fmImpl.NewClient(fleetManagerEndpoint, auth)
		Expect(err).ToNot(HaveOccurred())

		adminStaticToken := os.Getenv("STATIC_TOKEN_ADMIN")
		adminAuth, err := fmImpl.NewStaticAuth(context.Background(), fmImpl.StaticOption{StaticToken: adminStaticToken})
		Expect(err).ToNot(HaveOccurred())
		adminClient, err := fmImpl.NewClient(fleetManagerEndpoint, adminAuth)
		Expect(err).ToNot(HaveOccurred())
		fleetmanagerAdminClient = adminClient.AdminAPI()
	})

	Describe("should be created and deployed to Cluster 1", func() {
		fmt.Println(fleetmanagerAdminClient, fleetmanagerClient)
	})

})
