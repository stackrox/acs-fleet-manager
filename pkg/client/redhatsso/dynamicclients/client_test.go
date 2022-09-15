package dynamicclients

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/api"
	"github.com/stackrox/acs-fleet-manager/test/mocks"
	"github.com/stretchr/testify/assert"
)

var emptyCtx = context.Background()

func Test_rhSSOClient_CreateDynamicClient(t *testing.T) {
	type args struct {
		name         string
		orgID        string
		redirectURIs []string
	}
	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	clientID, clientSecret := server.GetInitialClientCredentials()

	tests := []struct {
		name        string
		realmConfig *iam.IAMRealmConfig
		args        args
		want        *api.AcsClientResponseData
		wantErr     bool
	}{
		{
			name: "should create a dynamic client",
			realmConfig: &iam.IAMRealmConfig{
				Realm:            "redhat-external",
				ClientID:         clientID,
				ClientSecret:     clientSecret, // pragma: allowlist secret
				BaseURL:          server.BaseURL(),
				APIEndpointURI:   "/auth/realms/redhat-external",
				TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
			},
			args: args{
				name:         "name",
				orgID:        "orgId",
				redirectURIs: []string{},
			},
			want: &api.AcsClientResponseData{
				Name: "name",
			},
			wantErr: false,
		},
		{
			name: "should fail with authentication error",
			realmConfig: &iam.IAMRealmConfig{
				Realm:            "redhat-external",
				BaseURL:          server.BaseURL(),
				APIEndpointURI:   "/auth/realms/redhat-external",
				TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
			},
			args: args{
				name:         "name",
				orgID:        "orgId",
				redirectURIs: []string{},
			},
			want:    nil,
			wantErr: true,
		},
	}
	g := NewWithT(t)
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			apiClient := NewDynamicClientsAPI(tt.realmConfig)
			got, _, err := apiClient.CreateAcsClient(emptyCtx,
				api.AcsClientRequestData{
					Name:         tt.args.name,
					OrgId:        tt.args.orgID,
					RedirectUris: tt.args.redirectURIs,
				})
			g.Expect(err != nil).To(Equal(tt.wantErr))
			if !tt.wantErr {
				g.Expect(got.Name).To(Equal(tt.want.Name))
			}
		})
	}
}

func Test_rhSSOClient_DeleteDynamicClient(t *testing.T) {
	server := mocks.NewMockServer()
	server.Start()
	defer server.Stop()
	clientID, clientSecret := server.GetInitialClientCredentials()

	realmConfig := &iam.IAMRealmConfig{
		Realm:            "redhat-external",
		ClientID:         clientID,
		ClientSecret:     clientSecret, // pragma: allowlist secret
		BaseURL:          server.BaseURL(),
		APIEndpointURI:   "/auth/realms/redhat-external",
		TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", server.BaseURL()),
	}

	apiClient := NewDynamicClientsAPI(realmConfig)
	dynamicClient, _, err := apiClient.CreateAcsClient(emptyCtx, api.AcsClientRequestData{
		Name:         "name",
		OrgId:        "orgId",
		RedirectUris: []string{},
	})
	assert.NoError(t, err)

	// 1. Delete existing dynamic client without an error
	_, err = apiClient.DeleteAcsClient(emptyCtx, dynamicClient.ClientId)
	assert.NoError(t, err)

	// 2. Attempt to delete non-existing dynamic client produces an error
	_, err = apiClient.DeleteAcsClient(emptyCtx, dynamicClient.ClientId)
	assert.Error(t, err)
}
