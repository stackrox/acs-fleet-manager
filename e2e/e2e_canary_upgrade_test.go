package e2e

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/operator"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/pkg/features"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	namespace = "acscs"
)

var _ = Describe("Fleetshard-sync Targeted Upgrade", func() {

	BeforeEach(func() {

		if !features.TargetedOperatorUpgrades.Enabled() || !features.StandaloneMode.Enabled() {
			Skip("Skipping canary upgrade test")
		}
	})

	if fmEndpointEnv := os.Getenv("FLEET_MANAGER_ENDPOINT"); fmEndpointEnv != "" {
		fleetManagerEndpoint = fmEndpointEnv
	}

	option := fleetmanager.OptionFromEnv()
	auth, err := fleetmanager.NewStaticAuth(fleetmanager.StaticOption{StaticToken: option.Static.StaticToken})
	Expect(err).ToNot(HaveOccurred())
	client, err := fleetmanager.NewClient(fleetManagerEndpoint, auth)
	Expect(err).ToNot(HaveOccurred())

	Describe("should run ACS operators", func() {

		It("should deploy single operator", func() {
			ctx := context.Background()
			oneOperatorVersionConfig := []operator.OperatorConfig{{
				GitRef: "4.1.1",
				Image:  "quay.io/rhacs-eng/stackrox-operator:4.1.1",
			}}
			err := updateOperatorConfig(ctx, oneOperatorVersionConfig)
			Expect(err).To(BeNil())

			It("should deploy operator with label selector 4.1.1", func() {
				Eventually(validateOperatorDeployment(ctx, "4.1.1", "quay.io/rhacs-eng/stackrox-operator:4.1.1")).
					WithTimeout(waitTimeout).
					WithPolling(defaultPolling).
					Should(Succeed())
			})

		})

		It("should deploy two operators with different versions", func() {
			ctx := context.Background()
			twoOperatorVersionConfig := []operator.OperatorConfig{{
				GitRef: "4.1.1",
				Image:  "quay.io/rhacs-eng/stackrox-operator:4.1.1",
			},
				{
					GitRef: "4.1.2",
					Image:  "quay.io/rhacs-eng/stackrox-operator:4.1.2",
				},
			}

			err := updateOperatorConfig(ctx, twoOperatorVersionConfig)
			Expect(err).To(BeNil())

			It("should deploy operator with label selector 4.1.1", func() {
				Eventually(validateOperatorDeployment(ctx, "4.1.1", "quay.io/rhacs-eng/stackrox-operator:4.1.1")).
					WithTimeout(waitTimeout).
					WithPolling(defaultPolling).
					Should(Succeed())
			})
			It("should deploy operator with label selector 4.1.2", func() {
				Eventually(validateOperatorDeployment(ctx, "4.1.2", "quay.io/rhacs-eng/stackrox-operator:4.1.2")).
					WithTimeout(waitTimeout).
					WithPolling(defaultPolling).
					Should(Succeed())
			})
			It("should contain only two operator deployments", func() {
				Eventually(getOperatorDeployments(ctx)).
					WithTimeout(waitTimeout).
					WithPolling(defaultPolling).
					Should(HaveLen(2))
			})
		})

		It("should delete the removed operator", func() {
			ctx := context.Background()
			operatorConfig := []operator.OperatorConfig{{
				GitRef: "4.1.2",
				Image:  "quay.io/rhacs-eng/stackrox-operator:4.1.2",
			}}
			err := updateOperatorConfig(ctx, operatorConfig)
			Expect(err).To(BeNil())

			It("should deploy operator with label selector 4.1.2", func() {
				Eventually(validateOperatorDeployment(ctx, "4.1.2", "quay.io/rhacs-eng/stackrox-operator:4.1.2")).
					WithTimeout(waitTimeout).
					WithPolling(defaultPolling).
					Should(Succeed())
			})
			It("should contain only one operator deployments", func() {
				Eventually(getOperatorDeployments(ctx)).
					WithTimeout(waitTimeout).
					WithPolling(defaultPolling).
					Should(HaveLen(1))
			})

		})

	})

	Describe("should upgrade the central", func() {

		var createdCentral *public.CentralRequest
		var centralNamespace string

		It("creates central", func() {
			centralName := newCentralName()
			request := public.CentralRequestPayload{
				CloudProvider: dpCloudProvider,
				MultiAz:       true,
				Name:          centralName,
				Region:        dpRegion,
			}
			resp, _, err := client.PublicAPI().CreateCentral(context.Background(), true, request)
			Expect(err).To(BeNil())
			createdCentral = &resp
			Expect(err).To(BeNil())
			Expect(constants.CentralRequestStatusAccepted.String()).To(Equal(createdCentral.Status))

			centralNamespace, err = services.FormatNamespace(createdCentral.Id)

			ctx := context.Background()
			oneOperatorVersionConfig := []operator.OperatorConfig{{
				GitRef: "4.1.1",
				Image:  "quay.io/rhacs-eng/stackrox-operator:4.1.1",
			}}
			err = updateOperatorConfig(ctx, oneOperatorVersionConfig)
			Expect(err).To(BeNil())

			Eventually(func() error {
				centralCR, err := getCentralCR(ctx, createdCentral.Name, centralNamespace)
				if err != nil {
					return fmt.Errorf("failed finding central CR: %v", err)
				}
				if centralCR.GetLabels()["rhacs.redhat.com/version-selector"] != "4.1.1" {
					return fmt.Errorf("central CR does not have 4.1.1 version-selector")
				}
				return nil
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())
			Eventually(func() error {
				centralDeployment, err := getCentralDeployment(ctx, createdCentral.Name, centralNamespace)
				if err != nil {
					return fmt.Errorf("failed finding central deployment: %v", err)
				}
				if centralDeployment.Spec.Template.Spec.Containers[0].Image == "quay.io/rhacs-eng/main:4.1.1" {
					return fmt.Errorf("there is no central deployment with 4.1.1 image tag")
				}
				return nil
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())
		})

		It("upgrade central", func() {
			ctx := context.Background()
			oneOperatorVersionConfig := []operator.OperatorConfig{{
				GitRef: "4.1.2",
				Image:  "quay.io/rhacs-eng/stackrox-operator:4.1.2",
			}}
			err = updateOperatorConfig(ctx, oneOperatorVersionConfig)
			Expect(err).To(BeNil())

			Eventually(func() error {
				centralDeployment, err := getCentralDeployment(ctx, createdCentral.Name, centralNamespace)
				if err != nil {
					return fmt.Errorf("failed finding central deployment: %v", err)
				}
				if centralDeployment.Spec.Template.Spec.Containers[0].Image == "quay.io/rhacs-eng/main:4.1.2" {
					return fmt.Errorf("there is no central deployment with 4.1.2 image tag")
				}
				return nil
			}).WithTimeout(waitTimeout).WithPolling(defaultPolling).Should(Succeed())
		})
	})

})

func updateOperatorConfig(ctx context.Context, operatorConfigs []operator.OperatorConfig) error {
	configYAML, err := yaml.Marshal(operatorConfigs)
	if err != nil {
		return err
	}
	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      operator.ACSOperatorConfigMap,
		},
		Data: map[string]string{
			"operator-config.yaml": string(configYAML),
		},
	}

	err = k8sClient.Delete(ctx, configMap)
	if err != nil && !apiErrors.IsNotFound(err) {
		return err
	}
	err = k8sClient.Create(ctx, configMap)
	if err != nil {
		return err
	}

	return nil
}

func getOperatorDeployments(ctx context.Context) []appsv1.Deployment {
	deployments := &appsv1.DeploymentList{}
	labels := map[string]string{"app": "rhacs-operator"}
	err := k8sClient.List(ctx, deployments,
		ctrlClient.InNamespace(operator.ACSOperatorNamespace),
		ctrlClient.MatchingLabels(labels))
	if err != nil {
		return nil
	}
	return deployments.Items
}

func getCentralCR(ctx context.Context, name string, namespace string) (*v1alpha1.Central, error) {
	central := &v1alpha1.Central{}
	centralKey := ctrlClient.ObjectKey{Namespace: namespace, Name: name}
	err := k8sClient.Get(ctx, centralKey, central)
	if err != nil {
		return nil, err
	}
	return central, nil
}

func getCentralDeployment(ctx context.Context, name string, namespace string) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	centralKey := ctrlClient.ObjectKey{Namespace: namespace, Name: name}
	err := k8sClient.Get(ctx, centralKey, deployment)
	return deployment, err
}

func validateOperatorDeployment(ctx context.Context, gitRef string, image string) error {
	deploymentName := "rhacs-operator-" + gitRef
	labelSelectorEnv := "rhacs.redhat.com/version-selector=" + gitRef
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: operator.ACSOperatorNamespace,
		},
	}

	err := k8sClient.Get(ctx, ctrlClient.ObjectKey{Namespace: operator.ACSOperatorNamespace, Name: deploymentName}, deployment)
	if err != nil {
		return err
	}
	foundManager := false
	foundLabelSelector := false
	containers := deployment.Spec.Template.Spec.Containers
	for _, container := range containers {
		if container.Name == "manager" {
			foundManager = true
			if container.Image != image {
				return fmt.Errorf("incorrect operator image %s", container.Image)
			}
			for _, envVar := range container.Env {
				if envVar.Name == "CENTRAL_LABEL_SELECTOR" {
					foundLabelSelector = true
					if envVar.Value != labelSelectorEnv {
						return fmt.Errorf("incorrect CENTRAL_LABEL_SELECTOR %s", envVar.Value)
					}
				}
			}
			if !foundLabelSelector {
				return fmt.Errorf("environment variable CENTRAL_LABEL_SELECTOR not found")
			}
		}
	}
	if !foundManager {
		return fmt.Errorf("manager container not found")
	}
	return nil
}
