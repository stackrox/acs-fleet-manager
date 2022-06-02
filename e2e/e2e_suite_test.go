package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

var cfg *rest.Config
var k8sClient client.Client

//TODO: Deploy fleet-manager, fleetshard-sync and database into a cluster
var _ = BeforeSuite(func() {
	k8sClient = k8s.CreateClientOrDie()
	test := "test"
	Expect(test).To(Equal("test"))
})

var _ = AfterSuite(func() {

})

func TestFleetManager(t *testing.T) {

}
