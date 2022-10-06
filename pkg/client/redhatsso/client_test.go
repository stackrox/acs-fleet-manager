package redhatsso

import (
	"fmt"
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

func CreateServiceAccountForTests(server mocks.RedhatSSOMock, accountName, accountDescription, clientID, clientSecret string) serviceaccountsclient.ServiceAccountData {
	c := NewSSOClient(&iam.IAMConfig{}, &iam.IAMRealmConfig{
		ClientID:         clientID,
		ClientSecret:     clientSecret, // pragma: allowlist secret - dummy value
		Realm:            "redhat-external",
		APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
		TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
	})
	serviceAccount, _ := c.CreateServiceAccount(accountName, accountDescription)
	return serviceAccount
}

func Test_rhSSOClient_GetConfig(t *testing.T) {
	type fields struct {
		config *iam.IAMConfig
	}
	tests := []struct {
		name   string
		fields fields
		want   *iam.IAMConfig
	}{
		{
			name: "should return the clients keycloak config",
			fields: fields{
				config: &iam.IAMConfig{},
			},
			want: &iam.IAMConfig{},
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			c := &rhSSOClient{
				config: tt.fields.config,
			}
			g.Expect(c.GetConfig()).To(Equal(tt.want))
		})
	}
}

func Test_rhSSOClient_GetRealmConfig(t *testing.T) {
	type fields struct {
		realmConfig *iam.IAMRealmConfig
	}
	tests := []struct {
		name   string
		fields fields
		want   *iam.IAMRealmConfig
	}{
		{
			name: "should return the clients keycloak Realm config",
			fields: fields{
				realmConfig: &iam.IAMRealmConfig{},
			},
			want: &iam.IAMRealmConfig{},
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			c := &rhSSOClient{
				realmConfig: tt.fields.realmConfig,
			}
			g.Expect(c.GetRealmConfig()).To(Equal(tt.want))
		})
	}
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
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
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
					BaseURL:          server.BaseURL(),
					ClientID:         "",
					ClientSecret:     "",
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL())},
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
			c := NewSSOClient(tt.fields.config, tt.fields.realmConfig)
			got, err := c.GetServiceAccounts(tt.args.first, tt.args.max)
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
		clientID string
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
		want    *serviceaccountsclient.ServiceAccountData
		found   bool
		wantErr bool
	}{
		{
			name: "should return the service account with matching clientId",
			fields: fields{
				config: &iam.IAMConfig{},
				realmConfig: &iam.IAMRealmConfig{
					ClientID:         clientID,
					ClientSecret:     clientSecret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
			},
			args: args{
				clientID: *serviceAccount.ClientId,
			},
			want:    &serviceAccount,
			found:   true,
			wantErr: false,
		},
		{
			name: "should fail if it cannot find the service account",
			fields: fields{
				config: &iam.IAMConfig{},
				realmConfig: &iam.IAMRealmConfig{
					ClientID:         clientID,
					ClientSecret:     clientSecret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
			},
			args: args{
				clientID: "wrong_clientId",
			},
			want:    nil,
			found:   false,
			wantErr: false,
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			c := NewSSOClient(tt.fields.config, tt.fields.realmConfig)
			got, httpStatus, err := c.GetServiceAccount(tt.args.clientID)
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(got).To(Equal(tt.want))
			g.Expect(httpStatus).To(Equal(tt.found))
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
					ClientSecret:     clientSecret,
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
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
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
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
			c := NewSSOClient(tt.fields.config, tt.fields.realmConfig)
			got, err := c.CreateServiceAccount(tt.args.name, tt.args.description)
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
		clientID string
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
					ClientSecret:     clientSecret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
			},
			args: args{
				clientID: *serviceAccount.ClientId,
			},
			wantErr: false,
		},
		{
			name: "should return an error if it fails to find service account for deletion",
			fields: fields{
				config: &iam.IAMConfig{},
				realmConfig: &iam.IAMRealmConfig{
					ClientID:         clientID,
					ClientSecret:     clientSecret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
			},
			args: args{
				clientID: "",
			},
			wantErr: true,
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			c := NewSSOClient(tt.fields.config, tt.fields.realmConfig)
			g.Expect(c.DeleteServiceAccount(tt.args.clientID) != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_rhSSOClient_UpdateServiceAccount(t *testing.T) {
	type fields struct {
		config      *iam.IAMConfig
		realmConfig *iam.IAMRealmConfig
	}
	type args struct {
		clientID    string
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
					ClientSecret:     clientSecret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
			},
			args: args{
				clientID:    *serviceAccount.ClientId,
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
					ClientSecret:     clientSecret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
			},
			args: args{
				clientID:    "",
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
			c := NewSSOClient(tt.fields.config, tt.fields.realmConfig)
			got, err := c.UpdateServiceAccount(tt.args.clientID, tt.args.name, tt.args.description)
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
					ClientSecret:     clientSecret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
			},
			args: args{
				id: *serviceAccount.ClientId,
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
					ClientSecret:     clientSecret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
			},
			args: args{
				id: "",
			},
			want: serviceaccountsclient.ServiceAccountData{
				Secret: serviceAccount.Secret,
			},
			wantErr: true,
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			c := NewSSOClient(tt.fields.config, tt.fields.realmConfig)
			got, err := c.RegenerateClientSecret(tt.args.id)
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(got).To(Not(Equal(tt.want)))
		})
	}
}
