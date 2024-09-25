package impl

import (
	"testing"

	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"github.com/stretchr/testify/assert"
)

func TestAuthOptions(t *testing.T) {
	tokenValue := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c" //pragma: allowlist secret
	t.Setenv("STATIC_TOKEN", tokenValue)
	t.Setenv("OCM_TOKEN", tokenValue)
	tokenFile := shared.BuildFullFilePath("./pkg/client/fleetmanager/impl/testdata/token")
	t.Setenv("FLEET_MANAGER_TOKEN_FILE", tokenFile)

	authOpt := OptionFromEnv()
	assert.Equal(t, "https://sso.redhat.com", authOpt.Sso.Endpoint)
	assert.Equal(t, "redhat-external", authOpt.Sso.Realm)
	assert.Equal(t, tokenValue, authOpt.Static.StaticToken)
	assert.Equal(t, tokenFile, authOpt.ServiceAccount.TokenFile)
}
