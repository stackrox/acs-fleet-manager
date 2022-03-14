package keycloak

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/pkg/errors"

	"github.com/Nerzal/gocloak/v8"
	"github.com/golang-jwt/jwt/v4"
	"github.com/onsi/gomega"
	"github.com/patrickmn/go-cache"
)

const (
	accessToken    = "accessToken"
	clientID       = "123"
	validIssuerURI = "testIssuerURI"
	jwtKeyFile     = "test/support/jwt_private_key.pem"
	jwtCAFile      = "test/support/jwt_ca.pem"
	issuerURL      = ""
)

func Test_kcClient_GetToken(t *testing.T) {
	authHelper, err := auth.NewAuthHelper(jwtKeyFile, jwtCAFile, issuerURL)
	if err != nil {
		t.Fatal(err)
	}

	acc, err := authHelper.NewAccount("username", "test-user", "", "org-id-0")
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		goCloakClient gocloak.GoCloak
		ctx           context.Context
		config        *KeycloakConfig
		realmConfig   *KeycloakRealmConfig
		cache         *cache.Cache
	}

	var goCloakToken gocloak.JWT
	cachedTK := fmt.Sprintf("%s%s", validIssuerURI, clientID)
	grantType := "grantType"
	Realm := "realmUno"
	JwksEndpointURI := "JwksEndpointURI"
	TokenEndpointURI := "TokenEndpointURI"
	tokenClaimType := ""
	claimsExpiredEXP := jwt.MapClaims{
		"typ": tokenClaimType,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Minute * time.Duration(-5)).Unix(),
	}
	jwtTokenExpired, _ := authHelper.CreateSignedJWT(acc, claimsExpiredEXP)
	tests := []struct {
		name         string
		fields       fields
		want         string
		wantErr      bool
		setupFn      func(f *fields)
		wantNewToken bool
	}{
		{
			name: "failed to get token",
			fields: fields{
				realmConfig: &KeycloakRealmConfig{
					ClientID:         clientID,
					GrantType:        grantType,
					ValidIssuerURI:   validIssuerURI,
					TokenEndpointURI: TokenEndpointURI,
					JwksEndpointURI:  JwksEndpointURI,
					Realm:            Realm,
				},
				goCloakClient: &GoCloakMock{
					GetTokenFunc: func(ctx context.Context, realm string, options gocloak.TokenOptions) (*gocloak.JWT, error) {
						return nil, errors.Errorf("failed to get token")
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			wantErr: true,
		},
		{
			name: "successfully get new access token when no token is in cache",
			fields: fields{
				realmConfig: &KeycloakRealmConfig{
					ClientID:         clientID,
					GrantType:        grantType,
					ValidIssuerURI:   validIssuerURI,
					TokenEndpointURI: TokenEndpointURI,
					JwksEndpointURI:  JwksEndpointURI,
					Realm:            Realm,
				},
				goCloakClient: &GoCloakMock{
					GetTokenFunc: func(ctx context.Context, realm string, options gocloak.TokenOptions) (*gocloak.JWT, error) {
						goCloakToken.AccessToken = accessToken
						return &goCloakToken, nil
					},
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
			},
			wantErr: false,
			want:    accessToken,
		},
		{
			name: "successfully create new token when token retrieved from cache is expired",
			setupFn: func(f *fields) {
				f.cache.Set(cachedTK, jwtTokenExpired, tokenLifeDuration)
			},
			fields: fields{
				realmConfig: &KeycloakRealmConfig{
					ClientID:         clientID,
					GrantType:        grantType,
					ValidIssuerURI:   validIssuerURI,
					TokenEndpointURI: TokenEndpointURI,
					JwksEndpointURI:  JwksEndpointURI,
					Realm:            Realm,
				},
				cache: cache.New(tokenLifeDuration, cacheCleanupInterval),
				goCloakClient: &GoCloakMock{
					GetTokenFunc: func(ctx context.Context, realm string, options gocloak.TokenOptions) (*gocloak.JWT, error) {
						goCloakToken.AccessToken = accessToken
						return &goCloakToken, nil
					},
				},
			},
			wantErr:      false,
			want:         accessToken,
			wantNewToken: true,
		},
	}
	for _, tt := range tests {
		gomega.RegisterTestingT(t)
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn(&tt.fields)
			}
			kc := &kcClient{
				kcClient:    tt.fields.goCloakClient,
				ctx:         tt.fields.ctx,
				config:      tt.fields.config,
				realmConfig: tt.fields.realmConfig,
				cache:       tt.fields.cache,
			}
			cachedToken, err := kc.GetToken()

			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			if cachedToken != "" && tt.wantNewToken {
				gomega.Expect(goCloakToken.AccessToken).To(gomega.Equal(tt.want))
			}
		})
	}
}

func Test_kcClient_IsClientExist(t *testing.T) {
	type fields struct {
		goCloakClient gocloak.GoCloak
		realmConfig   *KeycloakRealmConfig
	}

	type args struct {
		requestClientId string
		accessToken     string
	}

	grantType := "grantType"
	Realm := "realmUno"
	JwksEndpointURI := "JwksEndpointURI"
	TokenEndpointURI := "TokenEndpointURI"
	otherClientID := "456"
	correctClientID := "123"
	correctInternalID := "correctID"
	otherInternalID := "otherID"

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "error when no client exists with request clientId",
			fields: fields{
				goCloakClient: &GoCloakMock{
					GetClientsFunc: func(ctx context.Context, accessToken, realm string, params gocloak.GetClientsParams) ([]*gocloak.Client, error) {
						return nil, errors.Errorf("no client exists with requested clientId")
					},
				},
				realmConfig: &KeycloakRealmConfig{
					ClientID:         clientID,
					GrantType:        grantType,
					ValidIssuerURI:   validIssuerURI,
					TokenEndpointURI: TokenEndpointURI,
					JwksEndpointURI:  JwksEndpointURI,
					Realm:            Realm,
				},
			},
			args: args{
				requestClientId: otherClientID,
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "success when correct internal ID is returned",
			fields: fields{
				goCloakClient: &GoCloakMock{
					GetClientsFunc: func(ctx context.Context, accessToken, realm string, params gocloak.GetClientsParams) ([]*gocloak.Client, error) {
						return []*gocloak.Client{
							{
								ClientID: &otherClientID,
								ID:       &otherInternalID,
							},
							{
								ClientID: &correctClientID,
								ID:       &correctInternalID,
							},
						}, nil
					},
				},
				realmConfig: &KeycloakRealmConfig{
					ClientID:         clientID,
					GrantType:        grantType,
					ValidIssuerURI:   validIssuerURI,
					TokenEndpointURI: TokenEndpointURI,
					JwksEndpointURI:  JwksEndpointURI,
					Realm:            Realm,
				},
			},
			args: args{
				requestClientId: correctClientID,
			},
			wantErr: false,
			want:    correctInternalID,
		},
	}
	for _, tt := range tests {
		gomega.RegisterTestingT(t)
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{
				kcClient:    tt.fields.goCloakClient,
				realmConfig: tt.fields.realmConfig,
			}
			internalId, err := kc.IsClientExist(tt.args.requestClientId, tt.args.accessToken)

			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			gomega.Expect(internalId).To(gomega.Equal(tt.want))

		})
	}
}

func Test_kcClient_GetClient(t *testing.T) {
	type fields struct {
		goCloakClient gocloak.GoCloak
		realmConfig   *KeycloakRealmConfig
	}

	type args struct {
		requestClientId string
		accessToken     string
	}

	grantType := "grantType"
	Realm := "realmUno"
	JwksEndpointURI := "JwksEndpointURI"
	TokenEndpointURI := "TokenEndpointURI"
	otherClientID := "456"
	correctClientID := "123"
	correctInternalID := "correctID"
	otherInternalID := "otherID"

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "error when no client exists with request clientId",
			fields: fields{
				goCloakClient: &GoCloakMock{
					GetClientsFunc: func(ctx context.Context, accessToken, realm string, params gocloak.GetClientsParams) ([]*gocloak.Client, error) {
						return nil, errors.Errorf("no client exists with requested clientId")
					},
				},
				realmConfig: &KeycloakRealmConfig{
					ClientID:         clientID,
					GrantType:        grantType,
					ValidIssuerURI:   validIssuerURI,
					TokenEndpointURI: TokenEndpointURI,
					JwksEndpointURI:  JwksEndpointURI,
					Realm:            Realm,
				},
			},
			args: args{
				requestClientId: otherClientID,
			},
			wantErr: true,
		},
		{
			name: "success when correct internal ID is returned",
			fields: fields{
				goCloakClient: &GoCloakMock{
					GetClientsFunc: func(ctx context.Context, accessToken, realm string, params gocloak.GetClientsParams) ([]*gocloak.Client, error) {
						return []*gocloak.Client{
							{
								ClientID: &otherClientID,
								ID:       &otherInternalID,
							},
							{
								ClientID: &correctClientID,
								ID:       &correctInternalID,
							},
						}, nil
					},
				},
				realmConfig: &KeycloakRealmConfig{
					ClientID:         clientID,
					GrantType:        grantType,
					ValidIssuerURI:   validIssuerURI,
					TokenEndpointURI: TokenEndpointURI,
					JwksEndpointURI:  JwksEndpointURI,
					Realm:            Realm,
				},
			},
			args: args{
				requestClientId: "123",
			},
			wantErr: false,
			want:    correctInternalID,
		},
	}
	for _, tt := range tests {
		gomega.RegisterTestingT(t)
		t.Run(tt.name, func(t *testing.T) {
			kc := &kcClient{
				kcClient:    tt.fields.goCloakClient,
				realmConfig: tt.fields.realmConfig,
			}
			client, err := kc.GetClient(tt.args.requestClientId, tt.args.accessToken)

			gomega.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			if client != nil {
				gomega.Expect(*client.ID).To(gomega.Equal(tt.want))
			}
		})
	}
}
