package cluster

import (
	"context"
	"reflect"
	"testing"

	addonsV1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/testutils"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	addonNamespace = "rhacs"
)

func Test_addonService_GetAddon(t *testing.T) {
	type fields struct {
		k8sClient ctrlClient.Client
	}
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    shared.Addon
		wantErr bool
	}{
		{
			name: "should retrieve the addon when it exists",
			args: args{
				id: "acs-fleetshard",
			},
			fields: fields{
				k8sClient: testutils.NewFakeClientBuilder(t,
					addon(
						"acs-fleetshard",
						"0.2.0",
						"quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac",
						"quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c",
					),
					addonParametersSecret("addon-acs-fleetshard-parameters", map[string][]byte{"acscsEnvironment": []byte("test")}),
				).Build(),
			},
			want: shared.Addon{
				ID:           "acs-fleetshard",
				Version:      "0.2.0",
				SourceImage:  "quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac",
				PackageImage: "quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c",
				Parameters: map[string]string{
					"acscsEnvironment": "test",
				},
			},
		},
		{
			name: "should return error when addon does not exist",
			args: args{
				id: "acs-fleetshard",
			},
			fields: fields{
				k8sClient: testutils.NewFakeClientBuilder(t).Build(), // no objects
			},
			wantErr: true,
		},
		{
			name: "should return error when no package secret",
			args: args{
				id: "acs-fleetshard",
			},
			fields: fields{
				k8sClient: testutils.NewFakeClientBuilder(t,
					addon(
						"acs-fleetshard",
						"0.2.0",
						"quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac",
						"quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c",
					),
				).Build(),
			},
			wantErr: true,
			want: shared.Addon{
				ID:           "acs-fleetshard",
				Version:      "0.2.0",
				SourceImage:  "quay.io/osd-addons/acs-fleetshard-index@sha256:71eaaccb4d3962043eac953fb3c19a6cc6a88b18c472dd264efc5eb3da4960ac",
				PackageImage: "quay.io/osd-addons/acs-fleetshard-package@sha256:3e4fc039662b876c83dd4b48a9608d6867a12ab4932c5b7297bfbe50ba8ee61c",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ad := &addonService{
				k8sClient: tt.fields.k8sClient,
			}
			got, err := ad.GetAddon(context.TODO(), tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAddon() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAddon() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func addon(name string, version string, addonImage string, packageImage string) ctrlClient.Object {
	return &addonsV1alpha1.Addon{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: addonsV1alpha1.AddonSpec{
			Install: addonsV1alpha1.AddonInstallSpec{
				OLMOwnNamespace: &addonsV1alpha1.AddonInstallOLMOwnNamespace{
					AddonInstallOLMCommon: addonsV1alpha1.AddonInstallOLMCommon{
						CatalogSourceImage: addonImage,
						Channel:            "stable",
						Namespace:          addonNamespace,
						PackageName:        name,
					},
				},
			},
			Version: version,
			AddonPackageOperator: &addonsV1alpha1.AddonPackageOperator{
				Image: packageImage,
			},
		},
	}
}

func addonParametersSecret(name string, parameters map[string][]byte) ctrlClient.Object {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: addonNamespace,
		},
		Data: parameters,
	}
}
