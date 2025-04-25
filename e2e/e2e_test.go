package e2e

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/gitops"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/acs-fleet-manager/e2e/dns"
	"github.com/stackrox/acs-fleet-manager/e2e/testutil"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	fmImpl "github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager/impl"
	"github.com/stackrox/rox/operator/api/v1alpha1"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func newCentralName() string {
	rnd := make([]byte, 8)
	_, err := rand.Read(rnd)

	if err != nil {
		panic(fmt.Errorf("reading random bytes for unique central name: %w", err))
	}
	rndString := hex.EncodeToString(rnd)

	return fmt.Sprintf("%s-%s", "e2e", rndString)
}

const (
	defaultPolling = 1 * time.Second
	skipRouteMsg   = "route resource is not known to test cluster"
)

var (
	statusAccepted = constants.CentralRequestStatusAccepted.String()
)

var _ = Describe("Central", Ordered, func() {
	var client *fleetmanager.Client
	var adminAPI fleetmanager.AdminAPI
	var privateAPI fleetmanager.PrivateAPI
	var notes []string
	var ctx = context.Background()
	var serviceAccountToken string

	BeforeAll(func() {
		serviceAccountToken = createServiceAccountToken(context.Background())
	})

	BeforeEach(func() {
		testutil.SkipIf(!runCentralTests, "Skipping Central tests")

		option := fmImpl.OptionFromEnv()
		auth, err := fmImpl.NewStaticAuth(context.Background(), fmImpl.StaticOption{StaticToken: option.Static.StaticToken})
		Expect(err).ToNot(HaveOccurred())
		client, err = fmImpl.NewClient(fleetManagerEndpoint, auth)
		Expect(err).ToNot(HaveOccurred())

		adminStaticToken := os.Getenv("STATIC_TOKEN_ADMIN")
		adminAuth, err := fmImpl.NewStaticAuth(context.Background(), fmImpl.StaticOption{StaticToken: adminStaticToken})
		Expect(err).ToNot(HaveOccurred())
		adminClient, err := fmImpl.NewClient(fleetManagerEndpoint, adminAuth)
		Expect(err).ToNot(HaveOccurred())
		adminAPI = adminClient.AdminAPI()

		privateAPIAuth, err := fmImpl.NewStaticAuth(context.Background(), fmImpl.StaticOption{
			StaticToken: serviceAccountToken,
		})
		Expect(err).ToNot(HaveOccurred())
		privateClient, err := fmImpl.NewClient(fleetManagerEndpoint, privateAPIAuth)
		Expect(err).ToNot(HaveOccurred())
		privateAPI = privateClient.PrivateAPI()

		GinkgoWriter.Printf("Current time: %s\n", time.Now().String())
		printNotes(notes)
	})

	Describe("should be created and deployed to k8s", Ordered, func() {

		var centralRequestID string
		var centralRequestName string
		var namespaceName string

		BeforeAll(func() {
			Expect(restoreDefaultGitopsConfig()).To(Succeed())
		})

		BeforeAll(func() {
			resp, _, err := client.PublicAPI().CreateCentral(ctx, true, public.CentralRequestPayload{
				CloudProvider: dpCloudProvider,
				MultiAz:       true,
				Name:          newCentralName(),
				Region:        dpRegion,
			})
			Expect(err).To(BeNil())
			centralRequestID = resp.Id
			centralRequestName = resp.Name
			notes = []string{
				fmt.Sprintf("Central name: %s", resp.Name),
				fmt.Sprintf("Central ID: %s", resp.Id),
			}
			printNotes(notes)
			namespaceName, err = services.FormatNamespace(centralRequestID)
			Expect(err).To(BeNil())
			Expect(resp.Status).To(Equal(statusAccepted))
		})

		It("should transition central request state to provisioning", func() {
			Eventually(testutil.AssertCentralRequestProvisioning(ctx, client, centralRequestID)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should create central namespace", func() {
			var ns corev1.Namespace
			Eventually(assertNamespaceExists(ctx, &ns, namespaceName)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
			_, tenantLabelFound := ns.Labels["rhacs.redhat.com/tenant"]
			Expect(tenantLabelFound).To(BeTrue())
		})

		It("should generate a central-encryption-key-chain secret", func() {
			Eventually(assertSecretExists(ctx, &corev1.Secret{}, namespaceName, "central-encryption-key-chain")).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should create central CR in its namespace on a managed cluster", func() {
			Eventually(assertCentralCRExists(ctx, &v1alpha1.Central{}, namespaceName, centralRequestName)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		// TODO: possible flake. Maybe this test will be executed after the routes are created
		It("should not expose URLs until the routes are created", func() {
			testutil.SkipIf(!routesEnabled, skipRouteMsg)
			var centralRequest public.CentralRequest
			Expect(testutil.GetCentralRequest(ctx, client, centralRequestID, &centralRequest)).
				To(Succeed())
			Expect(centralRequest.CentralUIURL).To(BeEmpty())
			Expect(centralRequest.CentralDataURL).To(BeEmpty())
		})

		It("should transition central request state to ready", func() {
			Eventually(testutil.AssertCentralRequestReady(ctx, client, centralRequestID)).
				WithTimeout(extendedWaitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should have created central routes", func() {
			testutil.SkipIf(!routesEnabled, skipRouteMsg)

			var centralRequest public.CentralRequest
			Expect(testutil.GetCentralRequest(ctx, client, centralRequestID, &centralRequest)).
				To(Succeed())

			centralUIURL, err := url.Parse(centralRequest.CentralUIURL)
			Expect(err).ToNot(HaveOccurred())
			Expect(centralUIURL.Scheme).To(Equal("https"))

			Eventually(assertRouteExists(ctx, namespaceName, openshiftRouteV1.TLSTerminationReencrypt, centralUIURL.Host)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())

			centralDataHost, centralDataPort, err := net.SplitHostPort(centralRequest.CentralDataURL)
			Expect(err).ToNot(HaveOccurred())
			Expect(centralDataPort).To(Equal("443"))

			Eventually(assertRouteExists(ctx, namespaceName, openshiftRouteV1.TLSTerminationPassthrough, centralDataHost)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should have created AWS Route53 records", func() {
			testutil.SkipIf(!dnsEnabled, testutil.SkipDNSMsg)

			var centralRequest public.CentralRequest
			Expect(testutil.GetCentralRequest(ctx, client, centralRequestID, &centralRequest)).
				To(Succeed())

			var reencryptIngress openshiftRouteV1.RouteIngress
			Eventually(testutil.AssertReencryptIngressRouteExist(context.Background(), routeService, centralRequest, &reencryptIngress)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())

			dnsRecordsLoader := dns.NewRecordsLoader(route53Client, centralRequest)

			Eventually(dnsRecordsLoader.LoadDNSRecords).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(HaveLen(len(dnsRecordsLoader.CentralDomainNames)), "Started at %s", time.Now())

			recordSets := dnsRecordsLoader.LastResult
			testutil.AssertDNSMatchesRouter(dnsRecordsLoader.CentralDomainNames, recordSets, &reencryptIngress)
		})

		It("should backup important secrets in FM database", func() {
			expectedSecrets := k8s.NewSecretBackup(k8sClient, false).GetWatchedSecrets()
			Eventually(assertStoredSecrets(ctx, privateAPI, centralRequestID, expectedSecrets)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should set ForceReconcile through gitops", func() {
			cfg := defaultGitopsConfig()
			cfg.TenantResources.Overrides = append(cfg.TenantResources.Overrides, gitops.TenantResourceOverride{
				InstanceIDs: []string{centralRequestID},
				Values:      `forceReconcile: true`,
			})
			Expect(putGitopsConfig(ctx, cfg)).To(Succeed())
		})

		It("should restore secrets and deployment on namespace delete", func() {
			// Using managedDB false here because e2e don't run with managed postgresql
			secretBackup := k8s.NewSecretBackup(k8sClient, false)
			expectedSecrets, err := secretBackup.CollectSecrets(ctx, namespaceName)
			Expect(err).ToNot(HaveOccurred())

			deleteNamespaceAndWaitForRecreation(ctx, namespaceName, k8sClient)

			actualSecrets := map[string]*corev1.Secret{}
			Eventually(func() (err error) {
				actualSecrets, err = secretBackup.CollectSecrets(ctx, namespaceName) // pragma: allowlist secret
				return err
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())

			assertEqualSecrets(actualSecrets, expectedSecrets)
		})

		It("should set central-tls OwnerReference after restore", func() {
			centralTLSSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      k8s.CentralTLSSecretName,
					Namespace: namespaceName,
				},
			}

			Eventually(func() (err error) {
				if err := k8sClient.Get(ctx, ctrlClient.ObjectKeyFromObject(centralTLSSecret), centralTLSSecret); err != nil {
					return err
				}

				if len(centralTLSSecret.GetObjectMeta().GetOwnerReferences()) == 0 {
					return fmt.Errorf("OwnerReference for %s is empty", k8s.CentralTLSSecretName)
				}

				return nil
			}).WithPolling(time.Second * 10).WithTimeout(defaultTimeout).Should(Succeed())

		})

		It("should delete and recreate secret backup for admin reset API", func() {
			secretBackup := k8s.NewSecretBackup(k8sClient, false)
			oldSecrets, err := secretBackup.CollectSecrets(ctx, namespaceName)
			Expect(err).ToNot(HaveOccurred())
			Expect(oldSecrets).ToNot(BeEmpty())

			// modify secrets to later test that the backup was updated succesfully
			for _, secret := range oldSecrets {
				secret.Data["test"] = []byte("modified")
				err := k8sClient.Update(ctx, secret)
				Expect(err).ToNot(HaveOccurred())
			}

			_, err = adminAPI.CentralRotateSecrets(ctx, centralRequestID, private.CentralRotateSecretsRequest{ResetSecretBackup: true})
			Expect(err).ToNot(HaveOccurred())

			// Wait for secrets to be backed up again
			Eventually(func() error {
				central, _, err := privateAPI.GetCentral(ctx, centralRequestID)
				Expect(err).ToNot(HaveOccurred())
				if len(central.Metadata.SecretsStored) == 0 {
					return errors.New("secrets backup is empty")
				}

				return nil
			}).
				WithTimeout(waitTimeout).
				WithPolling(10 * time.Second).
				Should(Succeed())

			deleteNamespaceAndWaitForRecreation(ctx, namespaceName, k8sClient)

			var newSecrets map[string]*corev1.Secret
			Eventually(func() error {
				secrets, err := secretBackup.CollectSecrets(ctx, namespaceName)
				if err != nil {
					return err
				}
				newSecrets = secrets // pragma: allowlist secret
				return nil
			}).WithTimeout(1 * time.Minute).WithPolling(10).Should(Succeed())

			assertEqualSecrets(newSecrets, oldSecrets)
		})

		It("should transition central to deprovisioning state", func() {
			Expect(deleteCentralByID(ctx, client, centralRequestID)).
				To(Succeed())
			Eventually(testutil.AssertCentralRequestDeprovisioning(ctx, client, centralRequestID)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should delete Central CR", func() {
			Eventually(assertCentralCRDeleted(ctx, namespaceName, centralRequestName)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should remove central namespace", func() {
			Eventually(assertNamespaceDeleted(ctx, namespaceName)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should delete external DNS entries", func() {
			testutil.SkipIf(!dnsEnabled, testutil.SkipDNSMsg)
			var centralRequest public.CentralRequest
			Expect(testutil.GetCentralRequest(ctx, client, centralRequestID, &centralRequest)).
				To(Succeed())
			dnsRecordsLoader := dns.NewRecordsLoader(route53Client, centralRequest)
			Eventually(dnsRecordsLoader.LoadDNSRecords).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(BeEmpty(), "Started at %s", time.Now())
		})

		AfterAll(func() {
			Expect(restoreDefaultGitopsConfig()).To(Succeed())
		})
	})

	Describe("should be created and deployed to k8s with admin API", Ordered, func() {
		var centralRequestID string
		var centralRequestName string
		var namespaceName string

		BeforeAll(func() {
			centralName := newCentralName()
			request := private.CentralRequestPayload{
				Name:          centralName,
				MultiAz:       true,
				CloudProvider: dpCloudProvider,
				Region:        dpRegion,
			}
			resp, _, err := adminAPI.CreateCentral(ctx, true, request)
			Expect(err).To(BeNil())
			notes = []string{
				fmt.Sprintf("Central name: %s", resp.Name),
				fmt.Sprintf("Central ID: %s", resp.Id),
			}
			centralRequestID = resp.Id
			centralRequestName = resp.Name
			namespaceName, err = services.FormatNamespace(centralRequestID)
			Expect(err).To(BeNil())
			Expect(resp.Status).To(Equal(statusAccepted))
		})

		It("should create central in its namespace on a managed cluster", func() {
			Eventually(assertCentralCRExists(ctx, &v1alpha1.Central{}, namespaceName, centralRequestName)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should transition central's state to ready", func() {
			Eventually(testutil.AssertCentralRequestReady(ctx, client, centralRequestID)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should transition central to deprovisioning state when deleting", func() {
			Expect(deleteCentralByID(ctx, client, centralRequestID)).
				To(Succeed())
			Eventually(testutil.AssertCentralRequestDeprovisioning(ctx, client, centralRequestID)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should delete central CR", func() {
			Eventually(assertCentralCRDeleted(ctx, namespaceName, centralRequestName)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should remove central namespace", func() {
			Eventually(assertNamespaceDeleted(ctx, namespaceName)).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
		})

		It("should delete external DNS entries", func() {
			testutil.SkipIf(!dnsEnabled, testutil.SkipDNSMsg)
			var centralRequest public.CentralRequest
			Expect(testutil.GetCentralRequest(ctx, client, centralRequestID, &centralRequest)).
				To(Succeed())
			dnsRecordsLoader := dns.NewRecordsLoader(route53Client, centralRequest)
			Eventually(dnsRecordsLoader.LoadDNSRecords).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(BeEmpty(), "Started at %s", time.Now())
		})

		It("should be restorable", func() {
			By("calling the admin restore API", func() {
				res, err := adminAPI.RestoreCentral(ctx, centralRequestID)
				Expect(err).To(Not(HaveOccurred()))
				Expect(res.StatusCode).To(Equal(200))
			})

			By("reaching ready state", func() {
				Eventually(testutil.AssertCentralRequestReady(ctx, client, centralRequestID)).
					WithTimeout(waitTimeout).
					WithPolling(defaultPolling).
					Should(Succeed())
			})
		})

		It("should delete restored resources", func() {

			By("calling the delete API", func() {
				Expect(deleteCentralByID(ctx, client, centralRequestID)).To(Succeed())
			})

			By("removing central namespace", func() {
				Eventually(assertNamespaceDeleted(ctx, namespaceName)).
					WithTimeout(waitTimeout).
					WithPolling(defaultPolling).
					Should(Succeed())
			})

			By("deleting external DNS entries", func() {
				testutil.SkipIf(!dnsEnabled, testutil.SkipDNSMsg)
				var centralRequest public.CentralRequest
				Expect(testutil.GetCentralRequest(ctx, client, centralRequestID, &centralRequest)).
					To(Succeed())
				dnsRecordsLoader := dns.NewRecordsLoader(route53Client, centralRequest)
				Eventually(dnsRecordsLoader.LoadDNSRecords).
					WithTimeout(waitTimeout).
					WithPolling(defaultPolling).
					Should(BeEmpty(), "Started at %s", time.Now())
			})
		})

	})

	Describe("should be deployed and can be force-deleted", Ordered, func() {
		var centralRequestID string
		var centralRequestName string
		var namespaceName string

		BeforeAll(func() {
			centralName := newCentralName()
			request := public.CentralRequestPayload{
				Name:          centralName,
				MultiAz:       true,
				CloudProvider: dpCloudProvider,
				Region:        dpRegion,
			}

			resp, _, err := client.PublicAPI().CreateCentral(ctx, true, request)
			Expect(err).To(BeNil())
			centralRequestID = resp.Id
			centralRequestName = resp.Name
			notes = []string{
				fmt.Sprintf("Central name: %s", centralRequestName),
				fmt.Sprintf("Central ID: %s", centralRequestID),
			}
			namespaceName, err = services.FormatNamespace(centralRequestID)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(statusAccepted))
		})

		var readyCentralRequest public.CentralRequest

		It("should transition central's state to ready", func() {
			Eventually(testutil.AssertCentralRequestReady(ctx, client, centralRequestID)).
				WithTimeout(extendedWaitTimeout).
				WithPolling(defaultPolling).
				Should(Succeed())
			Expect(testutil.GetCentralRequest(ctx, client, centralRequestID, &readyCentralRequest)).
				To(Succeed())
		})

		It("should be deletable in the control-plane database", func() {
			_, err := adminAPI.DeleteDbCentralById(ctx, centralRequestID)
			Expect(err).ToNot(HaveOccurred())
			_, err = adminAPI.DeleteDbCentralById(ctx, centralRequestID)
			Expect(err).To(HaveOccurred())
			central, _, err := client.PublicAPI().GetCentralById(ctx, centralRequestID)
			Expect(err).To(HaveOccurred())
			Expect(central.Id).To(BeEmpty())
		})

		// Cleaning up on data-plane side because we have skipped the regular deletion workflow taking care of this.
		It("can be cleaned up manually", func() {
			// (1) Delete the Central CR.
			centralRef := &v1alpha1.Central{
				ObjectMeta: metav1.ObjectMeta{
					Name:      centralRequestName,
					Namespace: namespaceName,
				},
			}
			Expect(k8sClient.Delete(ctx, centralRef)).ToNot(HaveOccurred())

			// (2) Delete the namespace and everything in it.
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			Expect(k8sClient.Delete(ctx, namespace)).ToNot(HaveOccurred())
		})

		It("should delete external DNS entries", func() {
			testutil.SkipIf(!dnsEnabled, testutil.SkipDNSMsg)
			dnsRecordsLoader := dns.NewRecordsLoader(route53Client, readyCentralRequest)
			Eventually(dnsRecordsLoader.LoadDNSRecords).
				WithTimeout(waitTimeout).
				WithPolling(defaultPolling).
				Should(BeEmpty(), "Started at %s", time.Now())
		})
	})
})

func printNotes(notes []string) {
	for _, note := range notes {
		GinkgoWriter.Println(note)
	}
}

func deleteNamespaceAndWaitForRecreation(ctx context.Context, namespaceName string, k8sClient ctrlClient.Client) {
	previousNamespace := corev1.Namespace{}
	Expect(assertNamespaceExists(ctx, &previousNamespace, namespaceName)()).
		To(Succeed())

	previousCreationTime := previousNamespace.CreationTimestamp
	Expect(k8sClient.Delete(ctx, &previousNamespace)).
		NotTo(HaveOccurred())

	Eventually(func() error {
		newNamespace := corev1.Namespace{}
		if err := k8sClient.Get(ctx, ctrlClient.ObjectKey{Name: namespaceName}, &newNamespace); err != nil {
			return err
		}
		if previousCreationTime.Equal(&newNamespace.CreationTimestamp) {
			return fmt.Errorf("namespace found but was not yet deleted")
		}
		return nil
	}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())
}

func assertEqualSecrets(actualSecrets, expectedSecrets map[string]*corev1.Secret) {
	Expect(actualSecrets).ToNot(BeEmpty())
	Expect(len(actualSecrets)).To(Equal(len(expectedSecrets)))
	for secretName := range expectedSecrets { // pragma: allowlist secret
		actualData := actualSecrets[secretName].Data
		expectedData := expectedSecrets[secretName].Data
		Expect(actualData).To(Equal(expectedData))
	}
}

func ensureTestServiceAccountExists(ctx context.Context, sa *corev1.ServiceAccount) error {
	if err := k8sClient.Update(ctx, sa); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return err
		}
	} else {
		return nil
	}
	return k8sClient.Create(ctx, sa)
}

func createServiceAccountToken(ctx context.Context) string {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "e2e-tests",
		},
	}
	Expect(ensureTestServiceAccountExists(context.Background(), sa)).To(Succeed())

	token := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         []string{"acs-fleet-manager-private-api"},
			ExpirationSeconds: pointer.Int64(6 * 60 * 60), // 6 hours.
		},
	}
	err := k8sClient.SubResource("token").Create(ctx, sa, token)
	Expect(err).ToNot(HaveOccurred())
	return token.Status.Token
}
