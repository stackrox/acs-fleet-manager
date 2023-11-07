package gitops

import (
	"testing"

	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestDefaultCentral(t *testing.T) {
	p := getDummyCentralParams()
	central, err := renderDefaultCentral(p)
	assert.NoError(t, err)

	exposeEndpointEnabled := v1alpha1.ExposeEndpointEnabled
	autoScalingEnabled := v1alpha1.ScannerAutoScalingEnabled
	scannerComponentEnabled := v1alpha1.ScannerComponentEnabled

	wantCentral := v1alpha1.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Namespace,
			Labels: map[string]string{
				"rhacs.redhat.com/instance-type": p.InstanceType,
				"rhacs.redhat.com/org-id":        p.OrganizationID,
				"rhacs.redhat.com/tenant":        p.ID,
			},
			Annotations: map[string]string{
				"rhacs.redhat.com/org-name":             p.OrganizationName,
				"platform.stackrox.io/managed-services": "true",
			},
		},
		Spec: v1alpha1.CentralSpec{
			Central: &v1alpha1.CentralComponentSpec{
				AdminPasswordGenerationDisabled: pointers.Bool(true),
				Monitoring: &v1alpha1.Monitoring{
					ExposeEndpoint: &exposeEndpointEnabled,
				},
				DeploymentSpec: v1alpha1.DeploymentSpec{
					Resources: &v1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("4"),
							v1.ResourceMemory: resource.MustParse("8Gi"),
						},
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("2"),
							v1.ResourceMemory: resource.MustParse("4Gi"),
						},
					},
				},
			},
			Customize: &v1alpha1.CustomizeSpec{
				Annotations: map[string]string{
					"rhacs.redhat.com/org-name": p.OrganizationName,
				},
				Labels: map[string]string{
					"rhacs.redhat.com/instance-type": p.InstanceType,
					"rhacs.redhat.com/org-id":        p.OrganizationID,
					"rhacs.redhat.com/tenant":        p.ID,
				},
			},
			Scanner: &v1alpha1.ScannerComponentSpec{
				Analyzer: &v1alpha1.ScannerAnalyzerComponent{
					Scaling: &v1alpha1.ScannerAnalyzerScaling{
						AutoScaling: &autoScalingEnabled,
						MaxReplicas: pointers.Int32(3),
						MinReplicas: pointers.Int32(1),
						Replicas:    pointers.Int32(1),
					},
					DeploymentSpec: v1alpha1.DeploymentSpec{
						Resources: &v1.ResourceRequirements{
							Limits: v1.ResourceList{
								v1.ResourceCPU:    resource.MustParse("3"),
								v1.ResourceMemory: resource.MustParse("8Gi"),
							},
							Requests: v1.ResourceList{
								v1.ResourceCPU:    resource.MustParse("1.5"),
								v1.ResourceMemory: resource.MustParse("4Gi"),
							},
						},
					},
				},
				ScannerComponent: &scannerComponentEnabled,
				DB: &v1alpha1.DeploymentSpec{
					Resources: &v1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("2.5"),
							v1.ResourceMemory: resource.MustParse("4Gi"),
						},
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("1.25"),
							v1.ResourceMemory: resource.MustParse("2Gi"),
						},
					},
				},
				Monitoring: &v1alpha1.Monitoring{
					ExposeEndpoint: &exposeEndpointEnabled,
				},
			},
			Monitoring: &v1alpha1.GlobalMonitoring{
				OpenShiftMonitoring: &v1alpha1.OpenShiftMonitoring{
					Enabled: true,
				},
			},
		},
	}

	// compare yaml
	wantBytes, err := yaml.Marshal(wantCentral)
	assert.NoError(t, err)

	gotBytes, err := yaml.Marshal(central)
	assert.NoError(t, err)

	assert.YAMLEq(t, string(wantBytes), string(gotBytes))
}

func TestInternalCentral(t *testing.T) {
	p := getDummyCentralParams()
	p.IsInternal = true
	central, err := renderDefaultCentral(p)
	assert.NoError(t, err)

	exposeEndpointEnabled := v1alpha1.ExposeEndpointEnabled
	autoScalingEnabled := v1alpha1.ScannerAutoScalingEnabled
	scannerComponentEnabled := v1alpha1.ScannerComponentEnabled

	wantCentral := v1alpha1.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Namespace,
			Labels: map[string]string{
				"rhacs.redhat.com/instance-type": p.InstanceType,
				"rhacs.redhat.com/org-id":        p.OrganizationID,
				"rhacs.redhat.com/tenant":        p.ID,
			},
			Annotations: map[string]string{
				"rhacs.redhat.com/org-name":             p.OrganizationName,
				"platform.stackrox.io/managed-services": "true",
			},
		},
		Spec: v1alpha1.CentralSpec{
			Central: &v1alpha1.CentralComponentSpec{
				AdminPasswordGenerationDisabled: pointers.Bool(true),
				Monitoring: &v1alpha1.Monitoring{
					ExposeEndpoint: &exposeEndpointEnabled,
				},
				DeploymentSpec: v1alpha1.DeploymentSpec{
					Resources: &v1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("4"),
							v1.ResourceMemory: resource.MustParse("8Gi"),
						},
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("2"),
							v1.ResourceMemory: resource.MustParse("4Gi"),
						},
					},
				},
			},
			Customize: &v1alpha1.CustomizeSpec{
				Annotations: map[string]string{
					"rhacs.redhat.com/org-name": p.OrganizationName,
				},
				Labels: map[string]string{
					"rhacs.redhat.com/instance-type": p.InstanceType,
					"rhacs.redhat.com/org-id":        p.OrganizationID,
					"rhacs.redhat.com/tenant":        p.ID,
				},
			},
			Scanner: &v1alpha1.ScannerComponentSpec{
				Analyzer: &v1alpha1.ScannerAnalyzerComponent{
					Scaling: &v1alpha1.ScannerAnalyzerScaling{
						AutoScaling: &autoScalingEnabled,
						MaxReplicas: pointers.Int32(3),
						MinReplicas: pointers.Int32(1),
						Replicas:    pointers.Int32(1),
					},
					DeploymentSpec: v1alpha1.DeploymentSpec{
						Resources: &v1.ResourceRequirements{
							Limits: v1.ResourceList{
								v1.ResourceCPU:    resource.MustParse("3"),
								v1.ResourceMemory: resource.MustParse("8Gi"),
							},
							Requests: v1.ResourceList{
								v1.ResourceCPU:    resource.MustParse("1.5"),
								v1.ResourceMemory: resource.MustParse("4Gi"),
							},
						},
					},
				},
				ScannerComponent: &scannerComponentEnabled,
				DB: &v1alpha1.DeploymentSpec{
					Resources: &v1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("2.5"),
							v1.ResourceMemory: resource.MustParse("4Gi"),
						},
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("1.25"),
							v1.ResourceMemory: resource.MustParse("2Gi"),
						},
					},
				},
				Monitoring: &v1alpha1.Monitoring{
					ExposeEndpoint: &exposeEndpointEnabled,
				},
			},
			Monitoring: &v1alpha1.GlobalMonitoring{
				OpenShiftMonitoring: &v1alpha1.OpenShiftMonitoring{
					Enabled: false,
				},
			},
		},
	}

	// compare yaml
	wantBytes, err := yaml.Marshal(wantCentral)
	assert.NoError(t, err)

	gotBytes, err := yaml.Marshal(central)
	assert.NoError(t, err)

	assert.YAMLEq(t, string(wantBytes), string(gotBytes))
}
