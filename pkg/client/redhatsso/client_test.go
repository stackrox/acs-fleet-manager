package redhatsso

import (
	"fmt"
	"testing"

	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"

	. "github.com/onsi/gomega"
	"github.com/patrickmn/go-cache"
	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-go/serviceaccounts/apiv1internal/client"
	"github.com/stackrox/acs-fleet-manager/test/mocks"
)

const (
	accountName        = "serviceAccount"
	accountDescription = "fake service account"
)

func CreateServiceAccountForTests(accessToken string, server mocks.RedhatSSOMock, accountName string, accountDescription string) serviceaccountsclient.ServiceAccountData {
	c := &rhSSOClient{
		realmConfig: &iam.IAMRealmConfig{
			ClientID:         "",
			ClientSecret:     "",
			Realm:            "redhat-external",
			APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
			TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
		},
		configuration: &serviceaccountsclient.Configuration{
			DefaultHeader: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", accessToken),
				"Content-Type":  "application/json",
			},
			UserAgent: "OpenAPI-Generator/1.0.0/go",
			Debug:     false,
			Servers: serviceaccountsclient.ServerConfigurations{
				{
					URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
				},
			},
		},
	}
	serviceAccount, _ := c.CreateServiceAccount(accessToken, accountName, accountDescription)
	return serviceAccount
}

func TestNewSSOClient(t *testing.T) {
	type args struct {
		config      *iam.IAMConfig
		realmConfig *iam.IAMRealmConfig
	}
	tests := []struct {
		name string
		args args
		want SSOClient
	}{
		{
			name: "should successfully return a new sso client",
			args: args{
				config: &iam.IAMConfig{
					BaseURL: "base_url",
				},
				realmConfig: &iam.IAMRealmConfig{
					ClientID:     "Client_Id",
					ClientSecret: "ClientSecret",
				},
			},
			want: &rhSSOClient{
				config: &iam.IAMConfig{
					BaseURL: "base_url",
				},
				realmConfig: &iam.IAMRealmConfig{
					ClientID:     "Client_Id",
					ClientSecret: "ClientSecret",
				},
			},
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			ssoClient := NewSSOClient(tt.args.config, tt.args.realmConfig)
			g.Expect(ssoClient.GetConfig()).To(Equal(tt.want.GetConfig()))
			g.Expect(ssoClient.GetRealmConfig()).To(Equal(tt.want.GetRealmConfig()))
		})
	}
}

func Test_rhSSOClient_getConfiguration(t *testing.T) {
	type fields struct {
		config        *iam.IAMConfig
		realmConfig   *iam.IAMRealmConfig
		configuration *serviceaccountsclient.Configuration
		cache         *cache.Cache
	}
	type args struct {
		accessToken string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *serviceaccountsclient.Configuration
	}{
		{
			name: "should return the clients configuration",
			fields: fields{
				config:        &iam.IAMConfig{},
				realmConfig:   &iam.IAMRealmConfig{},
				configuration: nil,
				cache:         cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: "accessToken",
			},
			want: &serviceaccountsclient.Configuration{
				DefaultHeader: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", "accessToken"),
					"Content-Type":  "application/json",
				},
				UserAgent: "OpenAPI-Generator/1.0.0/go",
				Debug:     false,
				Servers: serviceaccountsclient.ServerConfigurations{
					{
						URL: "",
					},
				},
			},
		},
	}
	g := NewWithT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			g.Expect(c.getConfiguration(tt.args.accessToken)).To(Equal(tt.want))
		})
	}
}

func Test_rhSSOClient_getCachedToken(t *testing.T) {
	type fields struct {
		config        *iam.IAMConfig
		realmConfig   *iam.IAMRealmConfig
		configuration *serviceaccountsclient.Configuration
		cache         *cache.Cache
	}
	type args struct {
		tokenKey string
	}
	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	accessToken := server.GenerateNewAuthToken()

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
		setupFn func(f *fields)
	}{
		{
			name: "should return the token key if it is cached",
			fields: fields{
				config:        &iam.IAMConfig{},
				realmConfig:   &iam.IAMRealmConfig{},
				configuration: &serviceaccountsclient.Configuration{},
				cache:         cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				tokenKey: accessToken,
			},
			want:    accessToken,
			wantErr: false,
			setupFn: func(f *fields) {
				f.cache.Set(accessToken, accessToken, cacheCleanupInterval)
			},
		},
		{
			name: "should return an empty string and a error if token key is not cached",
			fields: fields{
				config:        &iam.IAMConfig{},
				realmConfig:   &iam.IAMRealmConfig{},
				configuration: &serviceaccountsclient.Configuration{},
				cache:         cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				tokenKey: "uncached-token-key",
			},
			want:    "",
			wantErr: true,
			setupFn: func(f *fields) {
				f.cache.Set(accessToken, server.GenerateNewAuthToken(), cacheCleanupInterval)
			},
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

		if tt.setupFn != nil {
			tt.setupFn(&tt.fields)
		}
		t.Run(tt.name, func(t *testing.T) {
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, err := c.getCachedToken(tt.args.tokenKey)
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(got).To(Equal(tt.want))
		})
	}
}

func Test_rhSSOClient_GetToken(t *testing.T) {
	type fields struct {
		config        *iam.IAMConfig
		realmConfig   *iam.IAMRealmConfig
		configuration *serviceaccountsclient.Configuration
		cache         *cache.Cache
	}
	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	accessToken := server.GenerateNewAuthToken()
	serviceAccount := CreateServiceAccountForTests(accessToken, server, accountName, accountDescription)
	server.SetBearerToken(*serviceAccount.Secret)

	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "should successfully return a clients token",
			fields: fields{
				config: &iam.IAMConfig{
					SsoBaseURL: server.BaseURL(),
				},
				realmConfig: &iam.IAMRealmConfig{
					ClientID:         *serviceAccount.ClientId,
					ClientSecret:     *serviceAccount.Secret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			want:    *serviceAccount.Secret,
			wantErr: false,
		},
		{
			name: "should return an error when it fails to retrieve the token key",
			fields: fields{
				config: &iam.IAMConfig{
					SsoBaseURL: server.BaseURL(),
				},
				realmConfig: &iam.IAMRealmConfig{
					ClientID:         "",
					ClientSecret:     "",
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			want:    "",
			wantErr: true,
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, err := c.GetToken()
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(got).To(Equal(tt.want))
		})

	}
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
		config        *iam.IAMConfig
		realmConfig   *iam.IAMRealmConfig
		configuration *serviceaccountsclient.Configuration
		cache         *cache.Cache
	}
	type args struct {
		accessToken string
		first       int
		max         int
	}
	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	accessToken := server.GenerateNewAuthToken()

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
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL())},
				configuration: &serviceaccountsclient.Configuration{
					DefaultHeader: map[string]string{
						"Authorization": fmt.Sprintf("Bearer %s", accessToken),
						"Content-Type":  "application/json",
					},
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				first:       0,
				max:         5,
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
				configuration: &serviceaccountsclient.Configuration{
					DefaultHeader: map[string]string{
						"Authorization": fmt.Sprintf("Bearer %s", accessToken),
						"Content-Type":  "application/json",
					},
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: "",
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				first:       0,
				max:         5,
			},
			want:    nil,
			wantErr: true,
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, err := c.GetServiceAccounts(tt.args.accessToken, tt.args.first, tt.args.max)
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(got).To(Equal(tt.want))
		})
	}
}

func Test_rhSSOClient_GetServiceAccount(t *testing.T) {
	type fields struct {
		config        *iam.IAMConfig
		realmConfig   *iam.IAMRealmConfig
		configuration *serviceaccountsclient.Configuration
		cache         *cache.Cache
	}
	type args struct {
		accessToken string
		clientID    string
	}

	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	accessToken := server.GenerateNewAuthToken()
	serviceAccount := CreateServiceAccountForTests(accessToken, server, accountName, accountDescription)

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
					ClientID:         *serviceAccount.ClientId,
					ClientSecret:     *serviceAccount.Secret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				clientID:    *serviceAccount.ClientId,
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
					ClientID:         *serviceAccount.ClientId,
					ClientSecret:     *serviceAccount.Secret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				clientID:    "wrong_clientId",
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
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, httpStatus, err := c.GetServiceAccount(tt.args.accessToken, tt.args.clientID)
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(got).To(Equal(tt.want))
			g.Expect(httpStatus).To(Equal(tt.found))
		})
	}
}

func Test_rhSSOClient_CreateServiceAccount(t *testing.T) {
	type fields struct {
		config        *iam.IAMConfig
		realmConfig   *iam.IAMRealmConfig
		configuration *serviceaccountsclient.Configuration
		cache         *cache.Cache
	}
	type args struct {
		accessToken string
		name        string
		description string
	}
	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	accessToken := server.GenerateNewAuthToken()
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
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
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
			name: "should fail to create the service account if wrong access token is given",
			fields: fields{
				config: &iam.IAMConfig{},
				realmConfig: &iam.IAMRealmConfig{
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: "wrong_access_token",
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
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, err := c.CreateServiceAccount(tt.args.accessToken, tt.args.name, tt.args.description)
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(got.Name).To(Equal(tt.want.Name))
			g.Expect(got.Description).To(Equal(tt.want.Description))
		})
	}
}

func Test_rhSSOClient_DeleteServiceAccount(t *testing.T) {
	type fields struct {
		config        *iam.IAMConfig
		realmConfig   *iam.IAMRealmConfig
		configuration *serviceaccountsclient.Configuration
		cache         *cache.Cache
	}
	type args struct {
		accessToken string
		clientID    string
	}

	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	accessToken := server.GenerateNewAuthToken()
	serviceAccount := CreateServiceAccountForTests(accessToken, server, accountName, accountDescription)

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
					ClientID:         *serviceAccount.ClientId,
					ClientSecret:     *serviceAccount.Secret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				clientID:    *serviceAccount.ClientId,
			},
			wantErr: false,
		},
		{
			name: "should return an error if it fails to find service account for deletion",
			fields: fields{
				config: &iam.IAMConfig{},
				realmConfig: &iam.IAMRealmConfig{
					ClientID:         *serviceAccount.ClientId,
					ClientSecret:     *serviceAccount.Secret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				clientID:    "",
			},
			wantErr: true,
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			g.Expect(c.DeleteServiceAccount(tt.args.accessToken, tt.args.clientID) != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_rhSSOClient_UpdateServiceAccount(t *testing.T) {
	type fields struct {
		config        *iam.IAMConfig
		realmConfig   *iam.IAMRealmConfig
		configuration *serviceaccountsclient.Configuration
		cache         *cache.Cache
	}
	type args struct {
		accessToken string
		clientID    string
		name        string
		description string
	}

	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	accessToken := server.GenerateNewAuthToken()
	serviceAccount := CreateServiceAccountForTests(accessToken, server, accountName, accountDescription)

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
					ClientID:         *serviceAccount.ClientId,
					ClientSecret:     *serviceAccount.Secret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				clientID:    *serviceAccount.ClientId,
				name:        "new name",
				description: "new description",
			},
			want: serviceaccountsclient.ServiceAccountData{
				Id:          serviceAccount.Id,
				ClientId:    serviceAccount.ClientId,
				Secret:      serviceAccount.Secret,
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
					ClientID:         *serviceAccount.ClientId,
					ClientSecret:     *serviceAccount.Secret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
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
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, err := c.UpdateServiceAccount(tt.args.accessToken, tt.args.clientID, tt.args.name, tt.args.description)
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(got).To(Equal(tt.want))
		})
	}
}

func Test_rhSSOClient_RegenerateClientSecret(t *testing.T) {
	type fields struct {
		config        *iam.IAMConfig
		realmConfig   *iam.IAMRealmConfig
		configuration *serviceaccountsclient.Configuration
		cache         *cache.Cache
	}
	type args struct {
		accessToken string
		id          string
	}

	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	accessToken := server.GenerateNewAuthToken()
	serviceAccount := CreateServiceAccountForTests(accessToken, server, accountName, accountDescription)

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
					ClientID:         *serviceAccount.ClientId,
					ClientSecret:     *serviceAccount.Secret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				id:          *serviceAccount.ClientId,
			},
			want: serviceaccountsclient.ServiceAccountData{
				Secret: &accessToken,
			},
			wantErr: false,
		},
		{
			name: "should return an error if it fails to find the service account to regenerate client secret",
			fields: fields{
				config: &iam.IAMConfig{},
				realmConfig: &iam.IAMRealmConfig{
					ClientID:         *serviceAccount.ClientId,
					ClientSecret:     *serviceAccount.Secret,
					Realm:            "redhat-external",
					APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
					TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
				},
				configuration: &serviceaccountsclient.Configuration{
					UserAgent: "OpenAPI-Generator/1.0.0/go",
					Debug:     false,
					Servers: serviceaccountsclient.ServerConfigurations{
						{
							URL: fmt.Sprintf("%s/auth/realms/redhat-external", server.BaseURL()),
						},
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			args: args{
				accessToken: accessToken,
				id:          "",
			},
			want: serviceaccountsclient.ServiceAccountData{
				Secret: &accessToken,
			},
			wantErr: true,
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			c := &rhSSOClient{
				config:        tt.fields.config,
				realmConfig:   tt.fields.realmConfig,
				configuration: tt.fields.configuration,
				cache:         tt.fields.cache,
			}
			got, err := c.RegenerateClientSecret(tt.args.accessToken, tt.args.id)
			g.Expect(err != nil).To(Equal(tt.wantErr))
			g.Expect(got).To(Not(Equal(tt.want)))
		})
	}
}
