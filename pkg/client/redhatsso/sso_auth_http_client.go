package redhatsso

import (
	"context"
	"net/http"

	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// NewSSOAuthHTTPClient returns http client which uses 2-legged OAuth2 flow to automatically get and refresh
// access token.
func NewSSOAuthHTTPClient(realmConfig *iam.IAMRealmConfig, scopes ...string) *http.Client {
	cfg := clientcredentials.Config{
		ClientID:     realmConfig.ClientID,
		ClientSecret: realmConfig.ClientSecret, // pragma: allowlist secret
		TokenURL:     realmConfig.TokenEndpointURI,
		Scopes:       scopes,
		AuthStyle:    oauth2.AuthStyleInParams,
	}
	ctx := context.Background()
	return oauth2.NewClient(ctx, cfg.TokenSource(ctx))
}
