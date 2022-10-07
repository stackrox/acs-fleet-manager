package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/serviceaccounts"
	"github.com/stretchr/testify/require"

	. "github.com/onsi/gomega"
	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-go/serviceaccounts/apiv1internal/client"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/test/mocks"
)

var emptyCtx = context.Background()

func getServiceAccountsAPI(baseURL, clientID, clientSecret string) serviceaccountsclient.ServiceAccountsApi {
	config := &iam.IAMRealmConfig{
		Realm:            "redhat-external",
		ClientID:         clientID,
		ClientSecret:     clientSecret, // pragma: allowlist secret - dummy value
		APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", baseURL),
		TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", baseURL),
	}

	return serviceaccounts.NewServiceAccountsAPI(config)
}

func Test_SSOClient_GetServiceAccounts(t *testing.T) {
	RegisterTestingT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	clientID, clientSecret := server.GetInitialClientCredentials()
	api := getServiceAccountsAPI(server.BaseURL(), clientID, clientSecret)

	// create 20 service accounts
	for i := 0; i < 20; i++ {
		_, _, err := api.CreateServiceAccount(emptyCtx).
			ServiceAccountCreateRequestData(createRequestData(fmt.Sprintf("test_%d", i), fmt.Sprintf("test account %d", i))).
			Execute()
		Expect(err).ToNot(HaveOccurred())
	}
	accounts, _, err := api.GetServiceAccounts(emptyCtx).
		First(0).
		Max(100).
		Execute()
	Expect(err).ToNot(HaveOccurred())
	Expect(accounts).To(HaveLen(20))
}

func Test_SSOClient_GetServiceAccount(t *testing.T) {
	RegisterTestingT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	clientID, clientSecret := server.GetInitialClientCredentials()
	api := getServiceAccountsAPI(server.BaseURL(), clientID, clientSecret)

	var serviceAccountList []serviceaccountsclient.ServiceAccountData
	// create 20 service accounts
	for i := 0; i < 3; i++ {
		serviceAccount, _, err := api.CreateServiceAccount(emptyCtx).
			ServiceAccountCreateRequestData(createRequestData(fmt.Sprintf("test_%d", i), fmt.Sprintf("test account %d", i))).
			Execute()
		Expect(err).ToNot(HaveOccurred())
		serviceAccountList = append(serviceAccountList, serviceAccount)
	}

	serviceAccount, resp, err := api.GetServiceAccount(emptyCtx, serviceAccountList[1].GetId()).Execute()
	Expect(err).ToNot(HaveOccurred())
	Expect(resp.StatusCode != http.StatusNotFound).To(BeTrue())
	Expect(serviceAccount).ToNot(BeNil())
	Expect(serviceAccount.GetSecret()).To(Equal(serviceAccountList[1].GetSecret()))
}

func Test_SSOClient_RegenerateSecret(t *testing.T) {
	RegisterTestingT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	clientID, clientSecret := server.GetInitialClientCredentials()
	api := getServiceAccountsAPI(server.BaseURL(), clientID, clientSecret)

	createdServiceAccount, _, err := api.CreateServiceAccount(emptyCtx).ServiceAccountCreateRequestData(
		createRequestData("accountName", "accountDescription")).Execute()
	Expect(err).ToNot(HaveOccurred())

	foundServiceAccount, resp, err := api.GetServiceAccount(emptyCtx, createdServiceAccount.GetId()).Execute()
	Expect(err).ToNot(HaveOccurred())
	Expect(resp.StatusCode != http.StatusNotFound).To(BeTrue())
	Expect(foundServiceAccount).ToNot(BeNil())
	Expect(foundServiceAccount.GetSecret()).To(Equal(createdServiceAccount.GetSecret()))

	updatedServiceAccount, _, err := api.ResetServiceAccountSecret(emptyCtx, foundServiceAccount.GetId()).Execute()
	Expect(err).ToNot(HaveOccurred())
	Expect(updatedServiceAccount).ToNot(BeNil())
	Expect(updatedServiceAccount.Id).To(Equal(foundServiceAccount.Id))
	Expect(updatedServiceAccount.Secret).ToNot(Equal(foundServiceAccount.Secret))
}

func Test_SSOClient_CreateServiceAccount(t *testing.T) {
	RegisterTestingT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	clientID, clientSecret := server.GetInitialClientCredentials()
	api := getServiceAccountsAPI(server.BaseURL(), clientID, clientSecret)

	serviceAccount, _, err := api.CreateServiceAccount(emptyCtx).
		ServiceAccountCreateRequestData(createRequestData("test_1", "test account 1")).
		Execute()
	require.NoError(t, err)
	Expect(*serviceAccount.Name).To(Equal("test_1"))
	Expect(*serviceAccount.Description).To(Equal("test account 1"))
}

func Test_SSOClient_DeleteServiceAccount(t *testing.T) {
	RegisterTestingT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	clientID, clientSecret := server.GetInitialClientCredentials()
	api := getServiceAccountsAPI(server.BaseURL(), clientID, clientSecret)

	// create 20 service accounts
	for i := 0; i < 20; i++ {
		_, _, err := api.CreateServiceAccount(emptyCtx).
			ServiceAccountCreateRequestData(createRequestData(fmt.Sprintf("test_%d", i), fmt.Sprintf("test account %d", i))).
			Execute()
		Expect(err).ToNot(HaveOccurred())
	}
	accounts, _, err := api.GetServiceAccounts(emptyCtx).
		First(0).
		Max(100).
		Execute()
	Expect(err).ToNot(HaveOccurred())
	Expect(accounts).To(HaveLen(20))
	_, err = api.DeleteServiceAccount(emptyCtx, accounts[5].GetId()).Execute()
	Expect(err).ToNot(HaveOccurred())
	accounts, _, err = api.GetServiceAccounts(emptyCtx).
		First(0).
		Max(100).
		Execute()
	Expect(err).ToNot(HaveOccurred())
	Expect(accounts).To(HaveLen(19))
}

func Test_SSOClient_UpdateServiceAccount(t *testing.T) {
	RegisterTestingT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	clientID, clientSecret := server.GetInitialClientCredentials()
	client := getServiceAccountsAPI(server.BaseURL(), clientID, clientSecret)

	// create 20 service accounts
	for i := 0; i < 20; i++ {
		_, _, err := client.CreateServiceAccount(emptyCtx).
			ServiceAccountCreateRequestData(createRequestData(fmt.Sprintf("test_%d", i), fmt.Sprintf("test account %d", i))).
			Execute()
		Expect(err).ToNot(HaveOccurred())
	}
	accounts, _, err := client.GetServiceAccounts(emptyCtx).
		First(0).
		Max(100).
		Execute()
	Expect(err).ToNot(HaveOccurred())
	Expect(accounts).To(HaveLen(20))

	updatedServiceAccount, _, err := client.UpdateServiceAccount(emptyCtx, accounts[5].GetId()).
		ServiceAccountRequestData(requestData("newName", "newName Description")).
		Execute()
	Expect(err).ToNot(HaveOccurred())
	Expect(*updatedServiceAccount.Name).To(Equal("newName"))
	Expect(*updatedServiceAccount.Description).To(Equal("newName Description"))
	Expect(*updatedServiceAccount.ClientId).To(Equal(*accounts[5].ClientId))
}

func requestData(updatedName string, updatedDescription string) serviceaccountsclient.ServiceAccountRequestData {
	return serviceaccountsclient.ServiceAccountRequestData{
		Name:        &updatedName,
		Description: &updatedDescription,
	}
}

func createRequestData(name, description string) serviceaccountsclient.ServiceAccountCreateRequestData {
	return serviceaccountsclient.ServiceAccountCreateRequestData{
		Name:        name,
		Description: &description,
	}
}
