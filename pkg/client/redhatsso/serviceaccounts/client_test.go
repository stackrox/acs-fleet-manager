package serviceaccounts

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"

	. "github.com/onsi/gomega"
	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-go/serviceaccounts/apiv1internal/client"
	"github.com/stackrox/acs-fleet-manager/test/mocks"
)

const (
	accountName        = "serviceAccount"
	accountDescription = "fake service account"
)

var emptyCtx = context.Background()

func CreateServiceAccountForTests(server mocks.RedhatSSOMock, accountName, accountDescription, clientID, clientSecret string) serviceaccountsclient.ServiceAccountData {
	api := NewServiceAccountsAPI(&iam.IAMRealmConfig{
		ClientID:         clientID,
		ClientSecret:     clientSecret, // pragma: allowlist secret - dummy value
		Realm:            "redhat-external",
		BaseURL:          server.BaseURL(),
		APIEndpointURI:   "/auth/realms/redhat-external",
		TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
	})

	serviceAccount, _, _ := api.CreateServiceAccount(emptyCtx).
		ServiceAccountCreateRequestData(serviceaccountsclient.ServiceAccountCreateRequestData{
			Name:        accountName,
			Description: &accountDescription,
		}).
		Execute()
	return serviceAccount
}

func Test_rhSSOClient_GetServiceAccounts(t *testing.T) {
	type fields struct {
		config      *iam.IAMConfig
		realmConfig *iam.IAMRealmConfig
	}
	type args struct {
		first int
		max   int
	}
	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	clientID, clientSecret := server.GetInitialClientCredentials()

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []serviceaccountsclient.ServiceAccountData
		wantErr bool
	}{
		{
			name: "should return a list of service accounts",
			fields: fields{
				config: &iam.IAMConfig{},
				realmConfig: &iam.IAMRealmConfig{
					Realm:            "redhat-external",
					ClientID:         clientID,
					ClientSecret:     clientSecret, // pragma: allowlist secret
					BaseURL:          server.BaseURL(),
					APIEndpointURI:   "/auth/realms/redhat-external",
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL())},
			},
			args: args{
				first: 0,
				max:   5,
			},
			want:    []serviceaccountsclient.ServiceAccountData{},
			wantErr: false,
		},
		{
			name: "should return an error when server URL is Missing",
			fields: fields{
				config: &iam.IAMConfig{},
				realmConfig: &iam.IAMRealmConfig{
					BaseURL:      server.BaseURL(),
					ClientID:     "",
					ClientSecret: "",
					Realm:        "redhat-external",
				},
			},
			args: args{
				first: 0,
				max:   5,
			},
			want:    nil,
			wantErr: true,
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			api := NewServiceAccountsAPI(tt.fields.realmConfig)
			got, _, err := api.GetServiceAccounts(emptyCtx).
				Max(int32(tt.args.max)).
				First(int32(tt.args.first)).
				Execute()
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(got).To(Equal(tt.want))
		})
	}
}

func Test_rhSSOClient_GetServiceAccount(t *testing.T) {
	type fields struct {
		config      *iam.IAMConfig
		realmConfig *iam.IAMRealmConfig
	}
	type args struct {
		id string
	}

	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	clientID, clientSecret := server.GetInitialClientCredentials()
	serviceAccount := CreateServiceAccountForTests(server, accountName, accountDescription, clientID, clientSecret)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    serviceaccountsclient.ServiceAccountData
		status  int
		wantErr bool
	}{
		{
			name: "should return the service account with matching clientId",
			fields: fields{
				config: &iam.IAMConfig{},
				realmConfig: &iam.IAMRealmConfig{
					ClientID:         clientID,
					ClientSecret:     clientSecret, // pragma: allowlist secret
					Realm:            "redhat-external",
					BaseURL:          server.BaseURL(),
					APIEndpointURI:   "/auth/realms/redhat-external",
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
			},
			args: args{
				id: *serviceAccount.Id,
			},
			want:    serviceAccount,
			status:  http.StatusOK,
			wantErr: false,
		},
		{
			name: "should fail if it cannot find the service account",
			fields: fields{
				config: &iam.IAMConfig{},
				realmConfig: &iam.IAMRealmConfig{
					ClientID:         clientID,
					ClientSecret:     clientSecret, // pragma: allowlist secret
					Realm:            "redhat-external",
					BaseURL:          server.BaseURL(),
					APIEndpointURI:   "/auth/realms/redhat-external",
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
			},
			args: args{
				id: "wrong_clientId",
			},
			want:    serviceaccountsclient.ServiceAccountData{},
			status:  http.StatusNotFound,
			wantErr: true,
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			api := NewServiceAccountsAPI(tt.fields.realmConfig)
			got, httpStatus, err := api.GetServiceAccount(emptyCtx, tt.args.id).Execute()
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(got).To(Equal(tt.want))
			g.Expect(httpStatus.StatusCode).To(Equal(tt.status))
		})
	}
}

func Test_rhSSOClient_CreateServiceAccount(t *testing.T) {
	type fields struct {
		config      *iam.IAMConfig
		realmConfig *iam.IAMRealmConfig
	}
	type args struct {
		name        string
		description string
	}
	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	clientID, clientSecret := server.GetInitialClientCredentials()
	accountName := "serviceAccount"
	accountDescription := "fake service account"

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    serviceaccountsclient.ServiceAccountData
		wantErr bool
	}{
		{
			name: "should successfully create the service account",
			fields: fields{
				config: &iam.IAMConfig{},
				realmConfig: &iam.IAMRealmConfig{
					Realm:            "redhat-external",
					ClientID:         clientID,
					ClientSecret:     clientSecret, // pragma: allowlist secret
					BaseURL:          server.BaseURL(),
					APIEndpointURI:   "/auth/realms/redhat-external",
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
			},
			args: args{
				name:        "serviceAccount",
				description: "fake service account",
			},
			want: serviceaccountsclient.ServiceAccountData{
				Name:        &accountName,
				Description: &accountDescription,
			},
			wantErr: false,
		},
		{
			name: "should fail to create the service account if wrong client credentials are given",
			fields: fields{
				config: &iam.IAMConfig{},
				realmConfig: &iam.IAMRealmConfig{
					Realm:            "redhat-external",
					ClientID:         "random client id",
					ClientSecret:     "random client secret", // pragma: allowlist secret
					BaseURL:          server.BaseURL(),
					APIEndpointURI:   "/auth/realms/redhat-external",
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
			},
			args: args{
				name:        "serviceAccount",
				description: "fake service account",
			},
			want: serviceaccountsclient.ServiceAccountData{
				Name:        nil,
				Description: nil,
			},
			wantErr: true,
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			api := NewServiceAccountsAPI(tt.fields.realmConfig)
			got, _, err := api.CreateServiceAccount(emptyCtx).
				ServiceAccountCreateRequestData(serviceaccountsclient.ServiceAccountCreateRequestData{
					Name:        tt.args.name,
					Description: &tt.args.description,
				}).
				Execute()
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(got.Name).To(Equal(tt.want.Name))
			g.Expect(got.Description).To(Equal(tt.want.Description))
		})
	}
}

func Test_rhSSOClient_DeleteServiceAccount(t *testing.T) {
	type fields struct {
		config      *iam.IAMConfig
		realmConfig *iam.IAMRealmConfig
	}
	type args struct {
		id string
	}

	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	clientID, clientSecret := server.GetInitialClientCredentials()
	serviceAccount := CreateServiceAccountForTests(server, accountName, accountDescription, clientID, clientSecret)

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "should successfully delete the service account",
			fields: fields{
				config: &iam.IAMConfig{},
				realmConfig: &iam.IAMRealmConfig{
					ClientID:         clientID,
					ClientSecret:     clientSecret, // pragma: allowlist secret
					Realm:            "redhat-external",
					BaseURL:          server.BaseURL(),
					APIEndpointURI:   "/auth/realms/redhat-external",
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
			},
			args: args{
				id: *serviceAccount.Id,
			},
			wantErr: false,
		},
		{
			name: "should return an error if it fails to find service account for deletion",
			fields: fields{
				config: &iam.IAMConfig{},
				realmConfig: &iam.IAMRealmConfig{
					ClientID:         clientID,
					ClientSecret:     clientSecret, // pragma: allowlist secret
					Realm:            "redhat-external",
					BaseURL:          server.BaseURL(),
					APIEndpointURI:   "/auth/realms/redhat-external",
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
			},
			args: args{
				id: "",
			},
			wantErr: true,
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			api := NewServiceAccountsAPI(tt.fields.realmConfig)
			_, err := api.DeleteServiceAccount(emptyCtx, tt.args.id).Execute()
			g.Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_rhSSOClient_UpdateServiceAccount(t *testing.T) {
	type fields struct {
		config      *iam.IAMConfig
		realmConfig *iam.IAMRealmConfig
	}
	type args struct {
		id          string
		name        string
		description string
	}

	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	clientID, clientSecret := server.GetInitialClientCredentials()
	serviceAccount := CreateServiceAccountForTests(server, accountName, accountDescription, clientID, clientSecret)

	name := "new name"
	description := "new description"

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    serviceaccountsclient.ServiceAccountData
		wantErr bool
	}{
		{
			name: "should successfully update the service account",
			fields: fields{
				config: &iam.IAMConfig{},
				realmConfig: &iam.IAMRealmConfig{
					ClientID:         clientID,
					ClientSecret:     clientSecret, // pragma: allowlist secret
					Realm:            "redhat-external",
					BaseURL:          server.BaseURL(),
					APIEndpointURI:   "/auth/realms/redhat-external",
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
			},
			args: args{
				id:          *serviceAccount.Id,
				name:        "new name",
				description: "new description",
			},
			want: serviceaccountsclient.ServiceAccountData{
				Id:          serviceAccount.Id,
				ClientId:    serviceAccount.ClientId,
				Secret:      serviceAccount.Secret, // pragma: allowlist secret
				Name:        &name,
				Description: &description,
				CreatedBy:   nil,
				CreatedAt:   nil,
			},
			wantErr: false,
		},
		{
			name: "should return an error if it fails to find the service account to update",
			fields: fields{
				config: &iam.IAMConfig{},
				realmConfig: &iam.IAMRealmConfig{
					ClientID:         clientID,
					ClientSecret:     clientSecret, // pragma: allowlist secret
					Realm:            "redhat-external",
					BaseURL:          server.BaseURL(),
					APIEndpointURI:   "/auth/realms/redhat-external",
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
			},
			args: args{
				id:          "",
				name:        "new name",
				description: "new description",
			},
			want:    serviceaccountsclient.ServiceAccountData{},
			wantErr: true,
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			api := NewServiceAccountsAPI(tt.fields.realmConfig)
			got, _, err := api.UpdateServiceAccount(emptyCtx, tt.args.id).
				ServiceAccountRequestData(serviceaccountsclient.ServiceAccountRequestData{
					Name:        &tt.args.name,
					Description: &tt.args.description,
				}).Execute()
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(got).To(Equal(tt.want))
		})
	}
}

func Test_rhSSOClient_RegenerateClientSecret(t *testing.T) {
	type fields struct {
		config      *iam.IAMConfig
		realmConfig *iam.IAMRealmConfig
	}
	type args struct {
		id string
	}

	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	clientID, clientSecret := server.GetInitialClientCredentials()
	serviceAccount := CreateServiceAccountForTests(server, accountName, accountDescription, clientID, clientSecret)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    serviceaccountsclient.ServiceAccountData
		wantErr bool
	}{
		{
			name: "should successfully regenerate the clients secret",
			fields: fields{
				config: &iam.IAMConfig{},
				realmConfig: &iam.IAMRealmConfig{
					ClientID:         clientID,
					ClientSecret:     clientSecret, // pragma: allowlist secret
					Realm:            "redhat-external",
					BaseURL:          server.BaseURL(),
					APIEndpointURI:   "/auth/realms/redhat-external",
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
			},
			args: args{
				id: *serviceAccount.Id,
			},
			want: serviceaccountsclient.ServiceAccountData{
				Secret: serviceAccount.Secret, // pragma: allowlist secret
			},
			wantErr: false,
		},
		{
			name: "should return an error if it fails to find the service account to regenerate client secret",
			fields: fields{
				config: &iam.IAMConfig{},
				realmConfig: &iam.IAMRealmConfig{
					ClientID:         clientID,
					ClientSecret:     clientSecret, // pragma: allowlist secret
					Realm:            "redhat-external",
					BaseURL:          server.BaseURL(),
					APIEndpointURI:   "/auth/realms/redhat-external",
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
			},
			args: args{
				id: "",
			},
			want: serviceaccountsclient.ServiceAccountData{
				Secret: serviceAccount.Secret, // pragma: allowlist secret
			},
			wantErr: true,
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			api := NewServiceAccountsAPI(tt.fields.realmConfig)
			got, _, err := api.ResetServiceAccountSecret(emptyCtx, tt.args.id).Execute()
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(got).To(Not(Equal(tt.want)))
		})
	}
}
