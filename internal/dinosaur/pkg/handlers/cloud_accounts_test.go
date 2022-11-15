package handlers

import (
	"context"
	"net/http"
	"testing"

	serviceErrors "github.com/stackrox/acs-fleet-manager/pkg/errors"

	"github.com/google/uuid"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stretchr/testify/assert"

	"github.com/pkg/errors"
)

const (
	JwtKeyFile = "test/support/jwt_private_key.pem"
	JwtCAFile  = "test/support/jwt_ca.pem"
)

func TestGetSuccess(t *testing.T) {
	testCloudAccount, err := v1.NewCloudAccount().
		CloudAccountID("cloudAccountID").
		CloudProviderID("cloudProviderID").
		Build()
	assert.NoError(t, err)
	c := ocm.ClientMock{
		GetOrganisationIDFromExternalIDFunc: func(externalID string) (string, error) {
			return "external-id", nil
		},
		GetCustomerCloudAccountsFunc: func(externalID string, quotaID []string) ([]*v1.CloudAccount, error) {
			return []*v1.CloudAccount{
				testCloudAccount,
			}, nil
		},
	}
	handler := NewCloudAccountsHandler(&c)

	authHelper, err := auth.NewAuthHelper(JwtKeyFile, JwtCAFile, "")
	assert.NoError(t, err)
	account, err := authHelper.NewAccount("username", "test-user", "", "org-id-0")
	assert.NoError(t, err)
	jwt, err := authHelper.CreateJWTWithClaims(account, nil)
	assert.NoError(t, err)
	authenticatedCtx := auth.SetTokenInContext(context.TODO(), jwt)
	r := &http.Request{}
	r = r.WithContext(authenticatedCtx)

	res, err := handler.actionFunc(r)()
	assert.Nil(t, err)
	cloudAccountsList, ok := res.(public.CloudAccountsList)
	assert.True(t, ok)

	assert.Len(t, cloudAccountsList.CloudAccounts, 1)
	assert.Equal(t, cloudAccountsList.CloudAccounts[0].CloudAccountId, testCloudAccount.CloudAccountID())
	assert.Equal(t, cloudAccountsList.CloudAccounts[0].CloudProviderId, testCloudAccount.CloudProviderID())
}

func TestGetNoOrgId(t *testing.T) {
	timesClientCalled := 0
	c := ocm.ClientMock{
		GetOrganisationIDFromExternalIDFunc: func(externalID string) (string, error) {
			timesClientCalled++
			return "external-id", nil
		},
		GetCustomerCloudAccountsFunc: func(externalID string, quotaID []string) ([]*v1.CloudAccount, error) {
			timesClientCalled++
			return []*v1.CloudAccount{}, nil
		},
	}
	handler := NewCloudAccountsHandler(&c)

	authHelper, err := auth.NewAuthHelper(JwtKeyFile, JwtCAFile, "")
	assert.NoError(t, err)
	builder := v1.NewAccount().
		ID(uuid.New().String()).
		Username("username").
		FirstName("Max").
		LastName("M").
		Email("example@redhat.com")
	account, err := builder.Build()
	assert.NoError(t, err)
	jwt, err := authHelper.CreateJWTWithClaims(account, nil)
	assert.NoError(t, err)
	authenticatedCtx := auth.SetTokenInContext(context.TODO(), jwt)
	r := &http.Request{}
	r = r.WithContext(authenticatedCtx)

	_, serviceErr := handler.actionFunc(r)()
	assert.Equal(t, serviceErr.Code, serviceErrors.ErrorForbidden)
	assert.Equal(t, 0, timesClientCalled)
}

func TestGetCannotGetOrganizationID(t *testing.T) {
	timesClientCalled := 0
	c := ocm.ClientMock{
		GetOrganisationIDFromExternalIDFunc: func(externalID string) (string, error) {
			return "", errors.New("test failure")
		},
		GetCustomerCloudAccountsFunc: func(externalID string, quotaID []string) ([]*v1.CloudAccount, error) {
			timesClientCalled++
			return []*v1.CloudAccount{}, nil
		},
	}
	handler := NewCloudAccountsHandler(&c)

	authHelper, err := auth.NewAuthHelper(JwtKeyFile, JwtCAFile, "")
	require.NoError(t, err)
	account, err := authHelper.NewAccount("username", "test-user", "", "org-id-0")
	assert.NoError(t, err)
	jwt, err := authHelper.CreateJWTWithClaims(account, nil)
	require.NoError(t, err)
	authenticatedCtx := auth.SetTokenInContext(context.TODO(), jwt)
	r := &http.Request{}
	r = r.WithContext(authenticatedCtx)

	_, serviceErr := handler.actionFunc(r)()
	assert.Equal(t, serviceErr.Code, serviceErrors.ErrorGeneral)
	assert.Equal(t, 0, timesClientCalled)
}
