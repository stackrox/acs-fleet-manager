package sso

import (
	"fmt"
	"testing"
	"time"

	pkgErr "github.com/pkg/errors"
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
		clusterId string
	}

	fakeId := "acs-fleetshard-agent-test-cluster-id"
	fakeClientId := "acs-fleetshard-agent-test-cluster-id"
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
							Id:          &fakeId,
							ClientId:    &fakeClientId,
							Secret:      &fakeClientSecret,
							Name:        &name,
							Description: &description,
							CreatedBy:   nil,
							CreatedAt:   &createdAt,
						}, nil
					},
					GetConfigFunc: func() *iam.IAMConfig {
						return iam.NewKeycloakConfig()
					},
				},
			},
			args: args{
				clusterId: "test-cluster-id",
			},
			want: &api.ServiceAccount{
				ID:           fakeClientId,
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
							Id:          &fakeId,
							ClientId:    &fakeClientId,
							Secret:      &fakeClientSecret,
							Name:        &name,
							Description: &description,
							CreatedBy:   nil,
							CreatedAt:   &createdAt,
						}, nil
					},
					GetConfigFunc: func() *iam.IAMConfig {
						return iam.NewKeycloakConfig()
					},
				},
			},
			args: args{
				clusterId: "test-cluster-id",
			},
			want: &api.ServiceAccount{
				ID:           fakeClientId,
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
			keycloakService := &redhatssoService{client: tt.fields.kcClient}
			got, err := keycloakService.RegisterAcsFleetshardOperatorServiceAccount(tt.args.clusterId)
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
		clusterId string
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
				clusterId: "test-cluster-id",
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
				clusterId: "test-cluster-id",
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
				clusterId: "test-cluster-id",
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
				clusterId: "test-cluster-id",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			keycloakService := &redhatssoService{client: tt.fields.kcClient}
			err := keycloakService.DeRegisterAcsFleetshardOperatorServiceAccount(tt.args.clusterId)
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestRedhatSSOService_DeleteServiceAccountInternal(t *testing.T) {
	t.Skip("DeleteServiceAccountInternal not yet implemented")

	type fields struct {
		kcClient redhatsso.SSOClient
	}
	tests := []struct {
		name       string
		disable    bool
		skipReason string
		fields     fields
		wantErr    bool
	}{
		{
			name: "returns error when failed to fetch token",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return "", pkgErr.New("token error")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "do not return an error when service account deleted successfully",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					DeleteServiceAccountFunc: func(accessToken string, clientId string) error {
						return nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "do not return an error when service account does not exists",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					DeleteServiceAccountFunc: func(accessToken string, clientId string) error {
						return fmt.Errorf("not found")
					},
				},
			},
			wantErr: false,
		},
		{
			name: "returns an error when failed to delete service account",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					DeleteServiceAccountFunc: func(accessToken string, clientId string) error {
						return fmt.Errorf("internal server error")
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			keycloakService := &redhatssoService{client: tt.fields.kcClient}
			err := keycloakService.DeleteServiceAccountInternal("account-id")
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}

}

func TestRedhatSSOService_CreateServiceAccountInternal(t *testing.T) {
	t.Skip("CreateServiceAccountInternal not yet implemented")
	tokenErr := pkgErr.New("token error")
	request := CompleteServiceAccountRequest{
		Owner:          "some-owner",
		OwnerAccountId: "owner-account-id",
		ClientId:       "some-client-id",
		Name:           "some-name",
		Description:    "some-description",
		OrgId:          "some-organisation-id",
	}
	type fields struct {
		kcClient redhatsso.SSOClient
	}
	tests := []struct {
		name                  string
		fields                fields
		wantErr               bool
		serviceAccountCreated bool
	}{
		{
			name: "returns error when failed to fetch token",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return "", tokenErr
					},
				},
			},
			wantErr:               true,
			serviceAccountCreated: false,
		},
		{
			name: "returns error when failed to create service account",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					//CreateProtocolMapperConfigFunc: func(s string) []gocloak.ProtocolMapperRepresentation {
					//	return []gocloak.ProtocolMapperRepresentation{}
					//},
					//ClientConfigFunc: func(client iam.ClientRepresentation) gocloak.Client {
					//	return gocloak.Client{}
					//},
					//CreateClientFunc: func(client gocloak.Client, accessToken string) (string, error) {
					//	return "", pkgErr.New("failed to create client")
					//},
					//GetClientFunc: func(clientId, accessToken string) (*gocloak.Client, error) {
					//	return nil, nil
					//},
				},
			},
			wantErr:               true,
			serviceAccountCreated: false,
		},
		{
			name: "succeed to create service account error when failed to create client",
			fields: fields{
				kcClient: &redhatsso.SSOClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					//GetClientFunc: func(clientId, accessToken string) (*gocloak.Client, error) {
					//	return nil, nil
					//},
					//CreateProtocolMapperConfigFunc: func(s string) []gocloak.ProtocolMapperRepresentation {
					//	return []gocloak.ProtocolMapperRepresentation{}
					//},
					//ClientConfigFunc: func(client iam.ClientRepresentation) gocloak.Client {
					//	return gocloak.Client{}
					//},
					//CreateClientFunc: func(client gocloak.Client, accessToken string) (string, error) {
					//	return "dsd", nil
					//},
					//GetClientSecretFunc: func(internalClientId, accessToken string) (string, error) {
					//	return "secret", nil
					//},
					//GetClientServiceAccountFunc: func(accessToken, internalClient string) (*gocloak.User, error) {
					//	return &gocloak.User{}, nil
					//},
					//UpdateServiceAccountUserFunc: func(accessToken string, serviceAccountUser gocloak.User) error {
					//	return nil
					//},
				},
			},
			wantErr:               false,
			serviceAccountCreated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			keycloakService := &redhatssoService{client: tt.fields.kcClient}
			serviceAccount, err := keycloakService.CreateServiceAccountInternal(request)
			Expect(err != nil).To(Equal(tt.wantErr))
			Expect(serviceAccount != nil).To(Equal(tt.serviceAccountCreated))
			if tt.serviceAccountCreated {
				Expect(serviceAccount.ClientSecret).To(Equal("secret"))
				Expect(serviceAccount.ClientID).To(Equal(request.ClientId))
				Expect(serviceAccount.ID).To(Equal("dsd"))
			}
		})
	}

}
