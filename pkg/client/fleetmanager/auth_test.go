package fleetmanager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthOptions(t *testing.T) {
	authOpt := &option{}

	tokenValue := "some-value"
	t.Setenv("STATIC_TOKEN", tokenValue)
	t.Setenv("OCM_TOKEN", tokenValue)

	opt := WithRhSSOOption(RhSsoOption{TokenFile: "some-file"})
	opt(authOpt)

	assert.Equal(t, "some-file", authOpt.Sso.TokenFile)
	assert.Empty(t, authOpt.Static.StaticToken)
	assert.Empty(t, authOpt.Ocm.RefreshToken)

	opt = WithOptionFromEnv()
	opt(authOpt)

	assert.Equal(t, "/run/secrets/rhsso-token/token", authOpt.Sso.TokenFile)
	assert.Equal(t, tokenValue, authOpt.Static.StaticToken)
	assert.Equal(t, tokenValue, authOpt.Ocm.RefreshToken)
}
