package integration

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/onsi/gomega"
	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-go/serviceaccounts/apiv1internal/client"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso"
	"github.com/stackrox/acs-fleet-manager/test/mocks"
)

func getClient(baseURL, clientID, clientSecret string) redhatsso.SSOClient {
	config := iam.IAMConfig{
		SsoBaseURL: baseURL,
		RedhatSSORealm: &iam.IAMRealmConfig{
			Realm:            "redhat-external",
			ClientID:         clientID,
			ClientSecret:     clientSecret, // pragma: allowlist secret - dummy value
			APIEndpointURI:   fmt.Sprintf("%s/auth/realms/redhat-external", baseURL),
			TokenEndpointURI: fmt.Sprintf("%s/auth/realms/redhat-external/protocol/openid-connect/token", baseURL),
		},
	}

	return redhatsso.NewSSOClient(&config, config.RedhatSSORealm)
}

func Test_SSOClient_GetServiceAccounts(t *testing.T) {
	RegisterTestingT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	clientID, clientSecret := server.GetInitialClientCredentials()
	client := getClient(server.BaseURL(), clientID, clientSecret)

	// create 20 service accounts
	for i := 0; i < 20; i++ {
		_, err := client.CreateServiceAccount(fmt.Sprintf("test_%d", i), fmt.Sprintf("test account %d", i))
		Expect(err).ToNot(HaveOccurred())
	}
	accounts, err := client.GetServiceAccounts(0, 100)
	Expect(err).ToNot(HaveOccurred())
	Expect(accounts).To(HaveLen(20))
}

func Test_SSOClient_GetServiceAccount(t *testing.T) {
	RegisterTestingT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	clientID, clientSecret := server.GetInitialClientCredentials()
	client := getClient(server.BaseURL(), clientID, clientSecret)

	var serviceAccountList []serviceaccountsclient.ServiceAccountData
	// create 20 service accounts
	for i := 0; i < 3; i++ {
		serviceAccount, err := client.CreateServiceAccount(fmt.Sprintf("test_%d", i), fmt.Sprintf("test account %d", i))
		Expect(err).ToNot(HaveOccurred())
		serviceAccountList = append(serviceAccountList, serviceAccount)
	}

	serviceAccount, found, err := client.GetServiceAccount(serviceAccountList[1].GetClientId())
	Expect(err).ToNot(HaveOccurred())
	Expect(found).To(BeTrue())
	Expect(serviceAccount).ToNot(BeNil())
	Expect(serviceAccount.GetSecret()).To(Equal(serviceAccountList[1].GetSecret()))
}

func Test_SSOClient_RegenerateSecret(t *testing.T) {
	RegisterTestingT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	clientID, clientSecret := server.GetInitialClientCredentials()
	client := getClient(server.BaseURL(), clientID, clientSecret)

	createdServiceAccount, err := client.CreateServiceAccount("accountName", "accountDescription")
	Expect(err).ToNot(HaveOccurred())

	foundServiceAccount, found, err := client.GetServiceAccount(createdServiceAccount.GetClientId())
	Expect(err).ToNot(HaveOccurred())
	Expect(found).To(BeTrue())
	Expect(foundServiceAccount).ToNot(BeNil())
	Expect(foundServiceAccount.GetSecret()).To(Equal(createdServiceAccount.GetSecret()))

	updatedServiceAccount, err := client.RegenerateClientSecret(foundServiceAccount.GetClientId())
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
	client := getClient(server.BaseURL(), clientID, clientSecret)

	serviceAccount, err := client.CreateServiceAccount("test_1", "test account 1")
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
	client := getClient(server.BaseURL(), clientID, clientSecret)

	// create 20 service accounts
	for i := 0; i < 20; i++ {
		_, err := client.CreateServiceAccount(fmt.Sprintf("test_%d", i), fmt.Sprintf("test account %d", i))
		Expect(err).ToNot(HaveOccurred())
	}
	accounts, err := client.GetServiceAccounts(0, 100)
	Expect(err).ToNot(HaveOccurred())
	Expect(accounts).To(HaveLen(20))
	err = client.DeleteServiceAccount(accounts[5].GetClientId())
	Expect(err).ToNot(HaveOccurred())
	accounts, err = client.GetServiceAccounts(0, 100)
	Expect(err).ToNot(HaveOccurred())
	Expect(accounts).To(HaveLen(19))
}

func Test_SSOClient_UpdateServiceAccount(t *testing.T) {
	RegisterTestingT(t)

	server := mocks.NewMockServer()
	server.Start()

	defer server.Stop()

	clientID, clientSecret := server.GetInitialClientCredentials()
	client := getClient(server.BaseURL(), clientID, clientSecret)

	// create 20 service accounts
	for i := 0; i < 20; i++ {
		_, err := client.CreateServiceAccount(fmt.Sprintf("test_%d", i), fmt.Sprintf("test account %d", i))
		Expect(err).ToNot(HaveOccurred())
	}
	accounts, err := client.GetServiceAccounts(0, 100)
	Expect(err).ToNot(HaveOccurred())
	Expect(accounts).To(HaveLen(20))

	updatedName := "newName"
	updatedDescription := "newName Description"

	updatedServiceAccount, err := client.UpdateServiceAccount(accounts[5].GetClientId(), updatedName, updatedDescription)
	Expect(err).ToNot(HaveOccurred())
	Expect(*updatedServiceAccount.Name).To(Equal(updatedName))
	Expect(*updatedServiceAccount.Description).To(Equal(updatedDescription))
	Expect(*updatedServiceAccount.ClientId).To(Equal(*accounts[5].ClientId))
}
