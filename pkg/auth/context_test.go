package auth

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
)

func TestContext_GetAccountIdFromClaims(t *testing.T) {
	tests := []struct {
		name   string
		claims ACSClaims
		want   string
	}{
		{
			name:   "Should return empty when tenantAccountIdClaim is empty",
			claims: ACSClaims{},
			want:   "",
		},
		{
			name: "Should return when tenantAccountIdClaim is not empty",
			claims: ACSClaims{
				tenantAccountIDClaim: "Test_account_id",
			},
			want: "Test_account_id",
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accountID, _ := tt.claims.GetAccountID()
			Expect(accountID).To(Equal(tt.want))
		})
	}
}

func TestContext_GetUserIdFromClaims(t *testing.T) {
	tests := []struct {
		name    string
		claims  ACSClaims
		want    string
		wantErr bool
	}{
		{
			name:    "Should return empty when tenantUserIDClaim and alternateUserIDClaim empty",
			claims:  ACSClaims{},
			want:    "",
			wantErr: true,
		},
		{
			name: "Should return when tenantUserIDClaim is not empty",
			claims: ACSClaims{
				tenantUserIDClaim: "12345678",
			},
			want: "12345678",
		},
		{
			name: "Should return when alternateUserIDClaim is not empty",
			claims: ACSClaims{
				alternateTenantUserIDClaim: "87654321",
			},
			want: "87654321",
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := tt.claims.GetUserID()
			Expect(userID).To(Equal(tt.want))
			if tt.wantErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).To(Not(HaveOccurred()))
			}
		})
	}
}

func TestContext_GetSubjectFromClaims(t *testing.T) {
	tests := []struct {
		name    string
		claims  ACSClaims
		want    string
		wantErr bool
	}{
		{
			name:    "Should return empty when tenantSubClaim empty",
			claims:  ACSClaims{},
			want:    "",
			wantErr: true,
		},
		{
			name: "Should return when tenantSubClaim is not empty",
			claims: ACSClaims{
				tenantSubClaim: "12345678",
			},
			want: "12345678",
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, err := tt.claims.GetSubject()
			Expect(sub).To(Equal(tt.want))
			if tt.wantErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).To(Not(HaveOccurred()))
			}
		})
	}
}

func TestContext_GetIsOrgAdminFromClaims(t *testing.T) {
	tests := []struct {
		name   string
		claims ACSClaims
		want   bool
	}{
		{
			name: "Should return true when tenantOrgAdminClaim is true",
			claims: ACSClaims{
				tenantOrgAdminClaim: true,
			},
			want: true,
		},
		{
			name: "Should return false when tenantOrgAdminClaim is false",
			claims: ACSClaims{
				tenantOrgAdminClaim: false,
			},
			want: false,
		},
		{
			name:   "Should return false when tenantOrgAdminClaim is false",
			claims: ACSClaims{},
			want:   false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(tt.claims.IsOrgAdmin()).To(Equal(tt.want))
		})
	}
}

func TestContext_GetUsernameFromClaims(t *testing.T) {
	tests := []struct {
		name   string
		claims ACSClaims
		want   string
	}{
		{
			name:   "Should return empty when tenantUsernameClaim and alternateUsernameClaim empty",
			claims: ACSClaims{},
			want:   "",
		},
		{
			name: "Should return when tenantUsernameClaim is not empty",
			claims: ACSClaims{
				tenantUsernameClaim: "Test Username",
			},
			want: "Test Username",
		},
		{
			name: "Should return when alternateUsernameClaim is not empty",
			claims: ACSClaims{
				alternateTenantUsernameClaim: "Test Alternate Username",
			},
			want: "Test Alternate Username",
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username, _ := tt.claims.GetUsername()
			Expect(username).To(Equal(tt.want))
		})
	}
}

func TestContext_GetOrgIdFromClaims(t *testing.T) {
	tests := []struct {
		name   string
		claims ACSClaims
		want   string
	}{
		{
			name:   "Should return empty when tenantIdClaim and alternateTenantIdClaim empty",
			claims: ACSClaims{},
			want:   "",
		},
		{
			name: "Should return tenantIdClaim when tenantIdClaim is not empty",
			claims: ACSClaims{
				tenantIDClaim: "Test Tenant ID",
			},
			want: "Test Tenant ID",
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orgID, _ := tt.claims.GetOrgID()
			Expect(orgID).To(Equal(tt.want))
		})
	}
}

func TestContext_GetIsAdminFromContext(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{
			name: "return false if isAdmin is false",
			ctx:  SetIsAdminContext(context.TODO(), false),
			want: false,
		},
		{
			name: "return true if isAdmin is true",
			ctx:  SetIsAdminContext(context.TODO(), true),
			want: true,
		},
		{
			name: "return false if isAdmin is nil",
			ctx:  SetFilterByOrganisationContext(context.TODO(), false),
			want: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(GetIsAdminFromContext(tt.ctx)).To(Equal(tt.want))
		})
	}
}

func TestContext_GetFilterByOrganisationFromContext(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{
			name: "return false if filterByOrganisation is false",
			ctx:  SetFilterByOrganisationContext(context.TODO(), false),
			want: false,
		},
		{
			name: "return true if filterByOrganisation is true",
			ctx:  SetFilterByOrganisationContext(context.TODO(), true),
			want: true,
		},
		{
			name: "return false if filterByOrganisaiton is nil",
			ctx:  SetIsAdminContext(context.TODO(), true),
			want: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(GetFilterByOrganisationFromContext(tt.ctx)).To(Equal(tt.want))
		})
	}
}
