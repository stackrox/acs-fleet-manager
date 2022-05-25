package sso

import (
	"fmt"
	"net/http"
	"testing"

	gocloak "github.com/Nerzal/gocloak/v11"
	. "github.com/onsi/gomega"

	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/keycloak"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"

	pkgErr "github.com/pkg/errors"
)

const (
	token        = "token"
	testClientID = "12221"
	secret       = "secret"
)

func TestKeycloakService_RegisterOSDClusterClientInSSO(t *testing.T) {
	tokenErr := pkgErr.New("token error")
	failedToCreateClientErr := pkgErr.New("failed to create client")

	type fields struct {
		kcClient keycloak.KcClient
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr *errors.ServiceError
	}{
		{
			name: "throws error when failed to fetch token",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", tokenErr
					},
				},
			},
			want:    "",
			wantErr: errors.NewWithCause(errors.ErrorGeneral, tokenErr, "error getting access token"),
		},
		{
			name: "fetch osd client secret from sso when client already exists",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return keycloak.NewKeycloakConfig()
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return testClientID, nil
					},
					GetClientSecretFunc: func(internalClientId string, accessToken string) (string, error) {
						return secret, nil
					},
				},
			},
			want:    secret,
			wantErr: nil,
		},
		{
			name: "successfully register a new sso client for the ACS cluster",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return keycloak.NewKeycloakConfig()
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "", nil
					},
					GetClientSecretFunc: func(internalClientId string, accessToken string) (string, error) {
						return secret, nil
					},
					CreateClientFunc: func(client gocloak.Client, accessToken string) (string, error) {
						return testClientID, nil
					},
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						testID := "12221"
						return gocloak.Client{
							ClientID: &testID,
						}
					},
				},
			},
			want:    secret,
			wantErr: nil,
		},
		{
			name: "failed to register sso client for the osd cluster",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return keycloak.NewKeycloakConfig()
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "", nil
					},
					GetClientSecretFunc: func(internalClientId string, accessToken string) (string, error) {
						return secret, nil
					},
					CreateClientFunc: func(client gocloak.Client, accessToken string) (string, error) {
						return "", failedToCreateClientErr
					},
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						testID := "12221"
						return gocloak.Client{
							ClientID: &testID,
						}
					},
				},
			},
			want:    "",
			wantErr: errors.NewWithCause(errors.ErrorFailedToCreateSSOClient, failedToCreateClientErr, "failed to create sso client"),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keycloakService := keycloakServiceProxy{
				accessTokenProvider: tt.fields.kcClient,
				service:             &masService{kcClient: tt.fields.kcClient},
			}
			got, err := keycloakService.RegisterClientInSSO("osd-cluster-12212", "https://oauth-openshift-cluster.fr")
			Expect(got).To(Equal(tt.want))
			Expect(err).To(Equal(tt.wantErr))
		})
	}

}

func TestKeycloakService_RegisterAcsFleetshardOperatorServiceAccount(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
	}
	type args struct {
		clusterId string
	}
	fakeRoleId := "1234"
	fakeClientId := "test-client-id"
	fakeClientSecret := "test-client-secret"
	fakeUserId := "test-user-id"
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					AddRealmRoleToUserFunc: func(accessToken string, userId string, role gocloak.Role) error {
						return nil
					},
					CreateClientFunc: func(client gocloak.Client, accessToken string) (string, error) {
						return fakeClientId, nil
					},
					GetClientFunc: func(clientId string, accessToken string) (*gocloak.Client, error) {
						return nil, nil
					},
					GetClientSecretFunc: func(internalClientId string, accessToken string) (string, error) {
						return fakeClientSecret, nil
					},
					GetClientServiceAccountFunc: func(accessToken string, internalClient string) (*gocloak.User, error) {
						return &gocloak.User{
							ID: &fakeUserId,
						}, nil
					},
					GetRealmRoleFunc: func(accessToken string, roleName string) (*gocloak.Role, error) {
						return &gocloak.Role{
							ID: &fakeRoleId,
						}, nil
					},
					UpdateServiceAccountUserFunc: func(accessToken string, serviceAccountUser gocloak.User) error {
						return nil
					},
					UserHasRealmRoleFunc: func(accessToken string, userId string, roleName string) (*gocloak.Role, error) {
						return nil, nil
					},
					CreateProtocolMapperConfigFunc: func(in1 string) []gocloak.ProtocolMapperRepresentation {
						return []gocloak.ProtocolMapperRepresentation{{}}
					},
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						return gocloak.Client{}
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return keycloak.NewKeycloakConfig()
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
				Name:         "acs-fleetshard-agent-test-cluster-id",
				Description:  "service account for agent on cluster test-cluster-id",
			},
			wantErr: false,
		},
		{
			name: "test registering serviceaccount for agent operator second time",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetClientFunc: func(clientId string, accessToken string) (*gocloak.Client, error) {
						return &gocloak.Client{
							ID: &fakeClientId,
						}, nil
					},
					GetClientSecretFunc: func(internalClientId string, accessToken string) (string, error) {
						return fakeClientSecret, nil
					},
					GetClientServiceAccountFunc: func(accessToken string, internalClient string) (*gocloak.User, error) {
						return &gocloak.User{
							ID: &fakeUserId,
							Attributes: &map[string][]string{
								acsClusterId: {"test-cluster-id"},
							},
						}, nil
					},
					GetRealmRoleFunc: func(accessToken string, roleName string) (*gocloak.Role, error) {
						return &gocloak.Role{
							ID: &fakeRoleId,
						}, nil
					},
					UserHasRealmRoleFunc: func(accessToken string, userId string, roleName string) (*gocloak.Role, error) {
						return &gocloak.Role{
							ID: &fakeRoleId,
						}, nil
					},
					CreateProtocolMapperConfigFunc: func(in1 string) []gocloak.ProtocolMapperRepresentation {
						return []gocloak.ProtocolMapperRepresentation{{}}
					},
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						return gocloak.Client{}
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return keycloak.NewKeycloakConfig()
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
				Name:         "acs-fleetshard-agent-test-cluster-id",
				Description:  "service account for agent on cluster test-cluster-id",
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keycloakService := keycloakServiceProxy{
				accessTokenProvider: tt.fields.kcClient,
				service:             &masService{kcClient: tt.fields.kcClient},
			}
			got, err := keycloakService.RegisterAcsFleetshardOperatorServiceAccount(tt.args.clusterId)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterAcsFleetshardOperatorServiceAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
			Expect(got).To(Equal(tt.want))
		})
	}
}

func TestKeycloakService_DeRegisterAcsFleetshardOperatorServiceAccount(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", fmt.Errorf("some errors")
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "testclietid", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "testclientid", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
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
			keycloakService := keycloakServiceProxy{
				accessTokenProvider: tt.fields.kcClient,
				service:             &masService{kcClient: tt.fields.kcClient},
			}
			err := keycloakService.DeRegisterAcsFleetshardOperatorServiceAccount(tt.args.clusterId)
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestKeycloakService_RegisterConnectorFleetshardOperatorServiceAccount(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
	}
	type args struct {
		clusterId string
		roleName  string
	}
	fakeRoleId := "1234"
	fakeClientId := "test-client-id"
	fakeClientSecret := "test-client-secret"
	fakeUserId := "test-user-id"
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					AddRealmRoleToUserFunc: func(accessToken string, userId string, role gocloak.Role) error {
						return nil
					},
					CreateClientFunc: func(client gocloak.Client, accessToken string) (string, error) {
						return fakeClientId, nil
					},
					GetClientFunc: func(clientId string, accessToken string) (*gocloak.Client, error) {
						return nil, nil
					},
					GetClientSecretFunc: func(internalClientId string, accessToken string) (string, error) {
						return fakeClientSecret, nil
					},
					GetClientServiceAccountFunc: func(accessToken string, internalClient string) (*gocloak.User, error) {
						return &gocloak.User{
							ID: &fakeUserId,
						}, nil
					},
					GetRealmRoleFunc: func(accessToken string, roleName string) (*gocloak.Role, error) {
						return &gocloak.Role{
							ID:   &fakeRoleId,
							Name: &roleName,
						}, nil
					},
					UpdateServiceAccountUserFunc: func(accessToken string, serviceAccountUser gocloak.User) error {
						return nil
					},
					UserHasRealmRoleFunc: func(accessToken string, userId string, roleName string) (*gocloak.Role, error) {
						return nil, nil
					},
					CreateProtocolMapperConfigFunc: func(in1 string) []gocloak.ProtocolMapperRepresentation {
						return []gocloak.ProtocolMapperRepresentation{{}}
					},
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						return gocloak.Client{}
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return keycloak.NewKeycloakConfig()
					},
				},
			},
			args: args{
				clusterId: "test-cluster-id",
				roleName:  "test-role-name",
			},
			want: &api.ServiceAccount{
				ID:           fakeClientId,
				ClientID:     "connector-fleetshard-agent-test-cluster-id",
				ClientSecret: fakeClientSecret,
				Name:         "connector-fleetshard-agent-test-cluster-id",
				Description:  "service account for agent on cluster test-cluster-id",
			},
			wantErr: false,
		},
		{
			name: "test registering serviceaccount for agent operator second time",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetClientFunc: func(clientId string, accessToken string) (*gocloak.Client, error) {
						return &gocloak.Client{
							ID: &fakeClientId,
						}, nil
					},
					GetClientSecretFunc: func(internalClientId string, accessToken string) (string, error) {
						return fakeClientSecret, nil
					},
					GetClientServiceAccountFunc: func(accessToken string, internalClient string) (*gocloak.User, error) {
						return &gocloak.User{
							ID: &fakeUserId,
							Attributes: &map[string][]string{
								connectorClusterId: {"test-cluster-id"},
							},
						}, nil
					},
					GetRealmRoleFunc: func(accessToken string, roleName string) (*gocloak.Role, error) {
						return &gocloak.Role{
							ID: &fakeRoleId,
						}, nil
					},
					UserHasRealmRoleFunc: func(accessToken string, userId string, roleName string) (*gocloak.Role, error) {
						return &gocloak.Role{
							ID: &fakeRoleId,
						}, nil
					},
					CreateProtocolMapperConfigFunc: func(in1 string) []gocloak.ProtocolMapperRepresentation {
						return []gocloak.ProtocolMapperRepresentation{{}}
					},
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						return gocloak.Client{}
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						return keycloak.NewKeycloakConfig()
					},
				},
			},
			args: args{
				clusterId: "test-cluster-id",
				roleName:  "test-role-name",
			},
			want: &api.ServiceAccount{
				ID:           fakeClientId,
				ClientID:     "connector-fleetshard-agent-test-cluster-id",
				ClientSecret: fakeClientSecret,
				Name:         "connector-fleetshard-agent-test-cluster-id",
				Description:  "service account for agent on cluster test-cluster-id",
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keycloakService := keycloakServiceProxy{
				accessTokenProvider: tt.fields.kcClient,
				service:             &masService{kcClient: tt.fields.kcClient},
			}
			got, err := keycloakService.RegisterConnectorFleetshardOperatorServiceAccount(tt.args.clusterId)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterConnectorFleetshardOperatorServiceAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
			Expect(got).To(Equal(tt.want))
		})
	}
}

func TestKeycloakService_DeRegisterConnectorFleetshardOperatorServiceAccount(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", fmt.Errorf("some errors")
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "testclietid", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "testclientid", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					IsClientExistFunc: func(clientId string, accessToken string) (string, error) {
						return "", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
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
			keycloakService := keycloakServiceProxy{
				accessTokenProvider: tt.fields.kcClient,
				service:             &masService{kcClient: tt.fields.kcClient},
			}
			err := keycloakService.DeRegisterConnectorFleetshardOperatorServiceAccount(tt.args.clusterId)
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestKeycloakService_DeleteServiceAccountInternal(t *testing.T) {
	tokenErr := pkgErr.New("token error")

	type fields struct {
		kcClient keycloak.KcClient
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "returns error when failed to fetch token",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", tokenErr
					},
				},
			},
			wantErr: true,
		},
		{
			name: "do not return an error when service account deleted successfully",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
						return nil
					},
					IsClientExistFunc: func(clientId, accessToken string) (string, error) {
						return "client-id", nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "do not return an error when service account does not exists",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
						return gocloak.APIError{
							Code: http.StatusNotFound,
						}
					},
					IsClientExistFunc: func(clientId, accessToken string) (string, error) {
						return "client-id", nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "returns an error when failed to delete service account",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					DeleteClientFunc: func(internalClientID, accessToken string) error {
						return gocloak.APIError{
							Code: http.StatusInternalServerError,
						}
					},
					IsClientExistFunc: func(clientId, accessToken string) (string, error) {
						return "client-id", nil
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			keycloakService := keycloakServiceProxy{
				accessTokenProvider: tt.fields.kcClient,
				service:             &masService{kcClient: tt.fields.kcClient},
			}
			err := keycloakService.DeleteServiceAccountInternal("account-id")
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}

}

func TestKeycloakService_CreateServiceAccountInternal(t *testing.T) {
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
		kcClient keycloak.KcClient
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
				kcClient: &keycloak.KcClientMock{
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
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					CreateProtocolMapperConfigFunc: func(s string) []gocloak.ProtocolMapperRepresentation {
						return []gocloak.ProtocolMapperRepresentation{}
					},
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						return gocloak.Client{}
					},
					CreateClientFunc: func(client gocloak.Client, accessToken string) (string, error) {
						return "", pkgErr.New("failed to create client")
					},
					GetClientFunc: func(clientId, accessToken string) (*gocloak.Client, error) {
						return nil, nil
					},
				},
			},
			wantErr:               true,
			serviceAccountCreated: false,
		},
		{
			name: "succeed to create service account error when failed to create client",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return "", nil
					},
					GetClientFunc: func(clientId, accessToken string) (*gocloak.Client, error) {
						return nil, nil
					},
					CreateProtocolMapperConfigFunc: func(s string) []gocloak.ProtocolMapperRepresentation {
						return []gocloak.ProtocolMapperRepresentation{}
					},
					ClientConfigFunc: func(client keycloak.ClientRepresentation) gocloak.Client {
						return gocloak.Client{}
					},
					CreateClientFunc: func(client gocloak.Client, accessToken string) (string, error) {
						return "dsd", nil
					},
					GetClientSecretFunc: func(internalClientId, accessToken string) (string, error) {
						return "secret", nil
					},
					GetClientServiceAccountFunc: func(accessToken, internalClient string) (*gocloak.User, error) {
						return &gocloak.User{}, nil
					},
					UpdateServiceAccountUserFunc: func(accessToken string, serviceAccountUser gocloak.User) error {
						return nil
					},
				},
			},
			wantErr:               false,
			serviceAccountCreated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			keycloakService := keycloakServiceProxy{
				accessTokenProvider: tt.fields.kcClient,
				service:             &masService{kcClient: tt.fields.kcClient},
			}
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

func TestKeycloakService_checkAllowedServiceAccountsLimits(t *testing.T) {
	type fields struct {
		kcClient keycloak.KcClient
	}
	type args struct {
		accessToken string
		maxAllowed  int
		orgId       string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Org ID present in the skip list, so no limits apply",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						config := keycloak.NewKeycloakConfig()
						config.ServiceAccounttLimitCheckSkipOrgIdList = []string{"01234", "56789"}
						return config
					},
					GetClientsFunc: func(accesstoken string, first int, max int, searchAttr string) ([]*gocloak.Client, error) {
						var clientArr []*gocloak.Client

						for i := 0; i < 2; i++ {
							client := gocloak.Client{}
							idStr := fmt.Sprintf("srvc-acct-%d", i)
							client.ClientID = &idStr
							clientArr = append(clientArr, &client)
						}
						return clientArr, nil
					},
				},
			},
			args: args{
				accessToken: "bearer: some-random-token",
				maxAllowed:  2,
				orgId:       "01234",
			},

			want:    true,
			wantErr: false,
		},
		{
			name: "Org ID not present in the skip list, limit allows creation of service account",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						config := keycloak.NewKeycloakConfig()
						config.ServiceAccounttLimitCheckSkipOrgIdList = []string{"01234", "56789"}
						return config
					},
					GetClientsFunc: func(accesstoken string, first int, max int, searchAttr string) ([]*gocloak.Client, error) {
						var clientArr []*gocloak.Client

						for i := 0; i < 2; i++ {
							client := gocloak.Client{}
							idStr := fmt.Sprintf("srvc-acct-%d", i)
							client.ClientID = &idStr
							clientArr = append(clientArr, &client)
						}
						return clientArr, nil
					},
				},
			},
			args: args{
				accessToken: "bearer: some-random-token",
				maxAllowed:  3,
				orgId:       "012345",
			},

			want:    true,
			wantErr: false,
		},
		{
			name: "Org ID not present in the skip list, limit disallows creation of service account",
			fields: fields{
				kcClient: &keycloak.KcClientMock{
					GetTokenFunc: func() (string, error) {
						return token, nil
					},
					GetConfigFunc: func() *keycloak.KeycloakConfig {
						config := keycloak.NewKeycloakConfig()
						config.ServiceAccounttLimitCheckSkipOrgIdList = []string{"01234", "56789"}
						return config
					},
					GetClientsFunc: func(accesstoken string, first int, max int, searchAttr string) ([]*gocloak.Client, error) {
						var clientArr []*gocloak.Client

						for i := 0; i < 2; i++ {
							client := gocloak.Client{}
							idStr := fmt.Sprintf("srvc-acct-%d", i)
							client.ClientID = &idStr
							clientArr = append(clientArr, &client)
						}
						return clientArr, nil
					},
				},
			},
			args: args{
				accessToken: "bearer: some-random-token",
				maxAllowed:  2,
				orgId:       "012345",
			},
			want:    false,
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := masService{
				kcClient: tt.fields.kcClient,
			}
			got, err := service.checkAllowedServiceAccountsLimits(tt.args.accessToken, tt.args.maxAllowed, tt.args.orgId)
			Expect(err != nil).To(Equal(tt.wantErr), "checkAllowedServiceAccountsLimits() error = %v, wantErr %v", err, tt.wantErr)
			Expect(tt.want).To(Equal(got))
		})
	}
}
