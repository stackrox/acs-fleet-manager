package fleetmanager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthOptions(t *testing.T) {
	tokenValue := "some-value"
	t.Setenv("STATIC_TOKEN", tokenValue)
	t.Setenv("OCM_TOKEN", tokenValue)
	authOpt := OptionFromEnv()
	assert.Equal(t, "/run/secrets/rhsso-token/token", authOpt.Sso.TokenFile)
	assert.Equal(t, tokenValue, authOpt.Static.StaticToken)
	assert.Equal(t, tokenValue, authOpt.Ocm.RefreshToken)
}
