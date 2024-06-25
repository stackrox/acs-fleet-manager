package auth

import (
	"fmt"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

func TestACSClaims_VerifyIssuer(t *testing.T) {
	const (
		validIssuer   = "https://valid-issuer"
		invalidIssuer = "https://invalid-issuer"
	)

	tests := map[string]struct {
		claims   ACSClaims
		issuer   string
		require  bool
		verified bool
	}{
		"should be verified with matching issuer": {
			claims: ACSClaims(jwt.MapClaims{
				"iss": validIssuer,
			}),
			issuer:   validIssuer,
			verified: true,
		},
		"should not be verified with non-matching issuer": {
			claims: ACSClaims(jwt.MapClaims{
				"iss": validIssuer,
			}),
			issuer: invalidIssuer,
		},
		"should not be verified with no issuer set but required set": {
			claims:  ACSClaims(jwt.MapClaims{}),
			issuer:  validIssuer,
			require: true,
		},
		"should be verified with no issuer set and issuer not required": {
			claims:   ACSClaims(jwt.MapClaims{}),
			issuer:   validIssuer,
			verified: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.verified, tt.claims.VerifyIssuer(tt.issuer, tt.require))
		})
	}
}

func TestACSClaims_GetUsername(t *testing.T) {
	const (
		claimUsername = "example-user"
	)
	tests := map[string]struct {
		claims   ACSClaims
		userName string
		error    bool
	}{
		"should yield username when claim username is set": {
			claims: ACSClaims(jwt.MapClaims{
				"username": claimUsername,
			}),
			userName: claimUsername,
		},
		"should yield username when claim preferred_username is set": {
			claims: ACSClaims(jwt.MapClaims{
				"preferred_username": claimUsername,
			}),
			userName: claimUsername,
		},
		"should yield error when no claim is set": {
			claims: ACSClaims(jwt.MapClaims{}),
			error:  true,
		},
		"should yield error when non-string value is set for any claim": {
			claims: ACSClaims(jwt.MapClaims{
				"preferred_username": 1234,
			}),
			error: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			userName, err := tt.claims.GetUsername()

			assert.Equal(t, tt.error, err != nil)
			assert.Equal(t, tt.userName, userName)
		})
	}
}

func TestACSClaims_GetAccountId(t *testing.T) {
	const (
		claimAccountID = "12345"
	)
	tests := map[string]struct {
		claims    ACSClaims
		accountID string
		error     bool
	}{
		"should yield account_id when claim is set": {
			claims: ACSClaims(jwt.MapClaims{
				"account_id": claimAccountID,
			}),
			accountID: claimAccountID,
		},
		"should yield error when no claim is set": {
			claims: ACSClaims(jwt.MapClaims{}),
			error:  true,
		},
		"should yield error when non-string value is set": {
			claims: ACSClaims(jwt.MapClaims{
				"account_id": 12345,
			}),
			error: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			accountID, err := tt.claims.GetAccountID()

			assert.Equal(t, tt.error, err != nil)
			assert.Equal(t, tt.accountID, accountID)
		})
	}
}

func TestACSClaims_GetOrgId(t *testing.T) {
	const (
		claimOrgID = "12345"
	)
	tests := map[string]struct {
		claims ACSClaims
		orgID  string
		error  bool
	}{
		"should yield org id when claim org_id is set": {
			claims: ACSClaims(jwt.MapClaims{
				"org_id": claimOrgID,
			}),
			orgID: claimOrgID,
		},
		"should yield org id when claim rh-org-id is set": {
			claims: ACSClaims(jwt.MapClaims{
				"rh-org-id": claimOrgID,
			}),
			orgID: claimOrgID,
		},
		"should yield error when no claim is set": {
			claims: ACSClaims(jwt.MapClaims{}),
			error:  true,
		},
		"should yield error when non-string value is set for any claim": {
			claims: ACSClaims(jwt.MapClaims{
				"org_id": 1234,
			}),
			error: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			orgID, err := tt.claims.GetOrgID()

			assert.Equal(t, tt.error, err != nil)
			assert.Equal(t, tt.orgID, orgID)
		})
	}
}

func TestACSClaims_GetUserId(t *testing.T) {
	const (
		claimUserID = "12345"
	)
	tests := map[string]struct {
		claims ACSClaims
		userID string
		error  bool
	}{
		"should yield sub when claim is set": {
			claims: ACSClaims(jwt.MapClaims{
				"sub": claimUserID,
			}),
			userID: claimUserID,
		},
		"should yield error when no claim is set": {
			claims: ACSClaims(jwt.MapClaims{}),
			error:  true,
		},
		"should yield error when non-string value is set": {
			claims: ACSClaims(jwt.MapClaims{
				"sub": 12345,
			}),
			error: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			userID, err := tt.claims.GetSubject()

			assert.Equal(t, tt.error, err != nil)
			assert.Equal(t, tt.userID, userID)
		})
	}
}

func TestACSClaims_GetTenantID(t *testing.T) {
	const (
		claimTenantID        = "12345"
		personalTokenSubject = "personal_token_sub"
	)
	var tests = map[string]struct {
		claims   ACSClaims
		tenantID string
		error    bool
	}{
		"should return tenantID when claim has subject with personal token": {
			claims: ACSClaims(jwt.MapClaims{
				"sub": personalTokenSubject,
			}),
			tenantID: personalTokenSubject,
		},
		"should return tenantID when claim has colon separated subject": {
			claims: ACSClaims(jwt.MapClaims{
				"sub": fmt.Sprintf("system:%s:%s%s:central", serviceAccountKey, rhacsNamespacePrefix, claimTenantID),
			}),
			tenantID: claimTenantID,
		},
		"should return error when subject is empty": {
			claims: ACSClaims(jwt.MapClaims{}),
			error:  true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tenantID, err := tt.claims.GetTenantID()

			assert.Equal(t, tt.error, err != nil)
			assert.Equal(t, tt.tenantID, tenantID)
		})
	}
}

func TestACSClaims_IsOrgAdmin(t *testing.T) {
	const (
		claimOrgAdmin = true
	)
	tests := map[string]struct {
		claims     ACSClaims
		isOrgAdmin bool
	}{
		"should yield org_admin when claim is set": {
			claims: ACSClaims(jwt.MapClaims{
				"is_org_admin": claimOrgAdmin,
			}),
			isOrgAdmin: claimOrgAdmin,
		},
		"should yield false when no claim is set": {
			claims: ACSClaims(jwt.MapClaims{}),
		},
		"should yield false when non-string value is set": {
			claims: ACSClaims(jwt.MapClaims{
				"is_org_admin": "true",
			}),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			isOrgAdmin := tt.claims.IsOrgAdmin()
			assert.Equal(t, tt.isOrgAdmin, isOrgAdmin)
		})
	}
}

func TestACSClaims_Audience(t *testing.T) {
	tests := map[string]struct {
		claims       ACSClaims
		expectValues []string
		expectError  bool
	}{
		"should parse the audience claim as string": {
			claims: ACSClaims(jwt.MapClaims{
				audienceClaim: "test",
			}),
			expectValues: []string{"test"},
		},
		"should parse the audience claim as an array of strings": {
			claims: ACSClaims(jwt.MapClaims{
				audienceClaim: []string{"test1", "test2"},
			}),
			expectValues: []string{"test1", "test2"},
		},
		"should parse the audience claim as an array of interfaces": {
			claims: ACSClaims(jwt.MapClaims{
				audienceClaim: []interface{}{"test"},
			}),
			expectValues: []string{"test"},
		},
		"should return error if there's no claim": {
			claims:      ACSClaims(jwt.MapClaims{}),
			expectError: true,
		},
		"should return empty slice if the claim is empty array": {
			claims: ACSClaims(jwt.MapClaims{
				audienceClaim: []string{},
			}),
			expectValues: []string{},
		},
		"should return empty slice if the claim is empty interface": {
			claims: ACSClaims(jwt.MapClaims{
				audienceClaim: []interface{}{},
			}),
			expectValues: []string{},
		},
		"should return error if can't parse the claim": {
			claims: ACSClaims(jwt.MapClaims{
				audienceClaim: 123,
			}),
			expectError: true,
		},
		"should not fail when part of the claim can't be parsed": {
			claims: ACSClaims(jwt.MapClaims{
				audienceClaim: []interface{}{123, "test"},
			}),
			expectValues: []string{"test"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			audience, err := tt.claims.GetAudience()
			assert.Equal(t, tt.expectError, err != nil)
			if !tt.expectError {
				assert.Equal(t, tt.expectValues, audience)
			}
		})
	}
}
