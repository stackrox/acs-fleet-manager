package sso

import (
	"fmt"
	"testing"
	"time"

	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-go/serviceaccounts/apiv1internal/client"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"

	. "github.com/onsi/gomega"
)

const (
	token = "token"
)

func TestRedhatSSOService_RegisterAcsFleetshardOperatorServiceAccount(t *testing.T) {
	type fields struct {
		kcClient redhatsso.SSOClient
	}
	type args struct {
		clusterID string
	}

	fakeID := "acs-fleetshard-agent-test-cluster-id"
	fakeClientID := "acs-fleetshard-agent-test-cluster-id"
	fakeClientSecret := "test-client-secret"
	createdAt := int64(0)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *api.ServiceAccount
		wantErr bool
	}{
		{
			name: "test registering serviceaccount for agent operator first time",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					CreateServiceAccountFunc: func(accessToken string, name string, description string) (serviceaccountsclient.ServiceAccountData, error) {
						return serviceaccountsclient.ServiceAccountData{
							Id:          &fakeID,
							ClientId:    &fakeClientID,
							Secret:      &fakeClientSecret,
							Name:        &name,
							Description: &description,
							CreatedBy:   nil,
							CreatedAt:   &createdAt,
						}, nil
					},
					GetConfigFunc: func() *iam.IAMConfig {
						return iam.NewIAMConfig()
					},
				},
			},
			args: args{
				clusterID: "test-cluster-id",
			},
			want: &api.ServiceAccount{
				ID:           fakeClientID,
				ClientID:     "acs-fleetshard-agent-test-cluster-id",
				ClientSecret: fakeClientSecret,
				Name:         "test-cluster-id",
				Description:  "service account for agent on cluster test-cluster-id",
				CreatedAt:    time.Unix(0, shared.SafeInt64(&createdAt)*int64(time.Millisecond)),
			},
			wantErr: false,
		},
		{
			name: "test registering serviceaccount for agent operator second time",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					CreateServiceAccountFunc: func(accessToken string, name string, description string) (serviceaccountsclient.ServiceAccountData, error) {
						return serviceaccountsclient.ServiceAccountData{
							Id:          &fakeID,
							ClientId:    &fakeClientID,
							Secret:      &fakeClientSecret,
							Name:        &name,
							Description: &description,
							CreatedBy:   nil,
							CreatedAt:   &createdAt,
						}, nil
					},
					GetConfigFunc: func() *iam.IAMConfig {
						return iam.NewIAMConfig()
					},
				},
			},
			args: args{
				clusterID: "test-cluster-id",
			},
			want: &api.ServiceAccount{
				ID:           fakeClientID,
				ClientID:     "acs-fleetshard-agent-test-cluster-id",
				ClientSecret: fakeClientSecret,
				Name:         "test-cluster-id",
				Description:  "service account for agent on cluster test-cluster-id",
				CreatedAt:    time.Unix(0, 0),
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iamService := &redhatssoService{client: tt.fields.kcClient}
			got, err := iamService.RegisterAcsFleetshardOperatorServiceAccount(tt.args.clusterID)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterAcsFleetshardOperatorServiceAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
			Expect(got).To(Equal(tt.want))
		})
	}
}

func TestRedhatSSOService_DeRegisterAcsFleetshardOperatorServiceAccount(t *testing.T) {
	type fields struct {
		kcClient redhatsso.SSOClient
	}
	type args struct {
		clusterID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "should receive an error when retrieving the token fails",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return "", fmt.Errorf("some errors")
					},
					DeleteServiceAccountFunc: func(accessToken string, clientId string) error {
						return fmt.Errorf("some error")
					},
				},
			},
			args: args{
				clusterID: "test-cluster-id",
			},
			wantErr: true,
		},
		{
			name: "should receive an error when service account deletion fails",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetServiceAccountFunc: func(accessToken string, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
						return nil, true, nil
					},
					DeleteServiceAccountFunc: func(accessToken string, clientId string) error {
						return fmt.Errorf("some error")
					},
				},
			},
			args: args{
				clusterID: "test-cluster-id",
			},
			wantErr: true,
		},
		{
			name: "should delete the service account",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetServiceAccountFunc: func(accessToken string, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
						return nil, true, nil
					},
					DeleteServiceAccountFunc: func(accessToken string, clientId string) error {
						return nil
					},
				},
			},
			args: args{
				clusterID: "test-cluster-id",
			},
			wantErr: false,
		},
		{
			name: "should not call delete if client doesn't exist",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetServiceAccountFunc: func(accessToken string, clientId string) (*serviceaccountsclient.ServiceAccountData, bool, error) {
						return nil, false, nil
					},
					DeleteServiceAccountFunc: func(accessToken string, clientId string) error {
						return fmt.Errorf("this should not be called")
					},
				},
			},
			args: args{
				clusterID: "test-cluster-id",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			iamService := &redhatssoService{client: tt.fields.kcClient}
			err := iamService.DeRegisterAcsFleetshardOperatorServiceAccount(tt.args.clusterID)
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestRedhatSSOService_GetRealmConfig(t *testing.T) {
	type fields struct {
		client redhatsso.SSOClient
	}
	tests := []struct {
		name   string
		fields fields
		want   *iam.IAMRealmConfig
	}{
		{
			name: "should return the realm config",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					GetRealmConfigFunc: func() *iam.IAMRealmConfig {
						return &iam.IAMRealmConfig{}
					},
				},
			},
			want: &iam.IAMRealmConfig{},
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			r := &redhatssoService{
				client: tt.fields.client,
			}
			g.Expect(r.GetRealmConfig()).To(Equal(tt.want))
		})
	}
}

func TestRedhatSSOService_GetConfig(t *testing.T) {
	type fields struct {
		client redhatsso.SSOClient
	}
	tests := []struct {
		name   string
		fields fields
		want   *iam.IAMConfig
	}{
		{
			name: "should return the config",
			fields: fields{
				client: &redhatsso.SSOClientMock{
					GetConfigFunc: func() *iam.IAMConfig {
						return &iam.IAMConfig{}
					},
				},
			},
			want: &iam.IAMConfig{},
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			r := &redhatssoService{
				client: tt.fields.client,
			}
			g.Expect(r.GetConfig()).To(Equal(tt.want))
		})
	}
}
