package mocks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/api"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-core/app-services-sdk-go/serviceaccountmgmt/apiv1/client"
)

// Unlimited ...
const (
	Unlimited = -1
)

// RedhatSSOMock ...
type RedhatSSOMock interface {
	Start()
	Stop()
	BaseURL() string
	GetInitialClientCredentials() (string, string)
	DeleteAllServiceAccounts()
	ServiceAccountsLimit() int
}

type redhatSSOMock struct {
	server               *httptest.Server
	authTokens           []string
	serviceAccounts      map[string]serviceaccountsclient.ServiceAccountData
	dynamicClients       map[string]api.AcsClientResponseData
	sessionAuthToken     string
	serviceAccountsLimit int
	initialClientID      string
	initialClientSecret  string
}

func (mockServer *redhatSSOMock) GetInitialClientCredentials() (string, string) {
	return mockServer.initialClientID, mockServer.initialClientSecret
}

type getTokenResponseMock struct {
	AccessToken      string `json:"access_token,omitempty"`
	ExpiresIn        int    `json:"expires_in,omitempty"`
	RefreshExpiresIn int    `json:"refresh_expires_in,omitempty"`
	TokenType        string `json:"token_type,omitempty"`
	NotBeforePolicy  int    `json:"not-before-policy,omitempty"`
	Scope            string `json:"scope,omitempty"`
}

var _ RedhatSSOMock = &redhatSSOMock{}

// MockServerOption ...
type MockServerOption func(mock *redhatSSOMock)

// WithServiceAccountLimit ...
func WithServiceAccountLimit(limit int) MockServerOption {
	return func(mock *redhatSSOMock) {
		mock.serviceAccountsLimit = limit
	}
}

// NewMockServer ...
func NewMockServer(options ...MockServerOption) RedhatSSOMock {
	mockServer := &redhatSSOMock{
		serviceAccounts:      make(map[string]serviceaccountsclient.ServiceAccountData),
		dynamicClients:       make(map[string]api.AcsClientResponseData),
		serviceAccountsLimit: Unlimited,
	}

	for _, option := range options {
		option(mockServer)
	}

	mockServer.init()
	return mockServer
}

// ServiceAccountsLimit ...
func (mockServer *redhatSSOMock) ServiceAccountsLimit() int {
	return mockServer.serviceAccountsLimit
}

// DeleteAllServiceAccounts ...
func (mockServer *redhatSSOMock) DeleteAllServiceAccounts() {
	mockServer.serviceAccounts = make(map[string]serviceaccountsclient.ServiceAccountData)
}

// Start ...
func (mockServer *redhatSSOMock) Start() {
	mockServer.server.Start()
}

// Stop ...
func (mockServer *redhatSSOMock) Stop() {
	mockServer.server.Close()
}

// BaseURL ...
func (mockServer *redhatSSOMock) BaseURL() string {
	return mockServer.server.URL
}

func (mockServer *redhatSSOMock) bearerAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		authorizationHeader := request.Header.Get("Authorization")
		if authorizationHeader != "" {
			for _, token := range mockServer.authTokens {
				if authorizationHeader == fmt.Sprintf("Bearer %s", token) {
					next.ServeHTTP(writer, request)
					return
				}
			}
		}

		http.Error(writer, "{\"error\":\"HTTP 401 Unauthorized\"}", http.StatusUnauthorized)
	})
}

func (mockServer *redhatSSOMock) serviceAccountAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		clientID := request.FormValue("client_id")
		clientSecret := request.FormValue("client_secret")
		// pragma: allowlist nextline secret
		if clientID == mockServer.initialClientID && mockServer.initialClientSecret == clientSecret {
			next.ServeHTTP(writer, request)
			return
		}

		http.Error(writer, "{\"error\":\"unauthorized_client\",\"error_description\":\"Invalid client secret\"}", http.StatusUnauthorized)
	})
}

func (mockServer *redhatSSOMock) init() {
	r := mux.NewRouter()
	bearerTokenAuthRouter := r.NewRoute().Subrouter()
	bearerTokenAuthRouter.Use(mockServer.bearerAuthMiddleware)
	serviceAccountAuthenticatedRouter := r.NewRoute().Subrouter()
	serviceAccountAuthenticatedRouter.Use(mockServer.serviceAccountAuthMiddleware)

	serviceAccountAuthenticatedRouter.HandleFunc("/auth/realms/redhat-external/protocol/openid-connect/token", mockServer.getTokenHandler).Methods("POST")

	bearerTokenAuthRouter.HandleFunc("/auth/realms/redhat-external/apis/service_accounts/v1", mockServer.createServiceAccountHandler).Methods("POST")
	bearerTokenAuthRouter.HandleFunc("/auth/realms/redhat-external/apis/service_accounts/v1", mockServer.getServiceAccountsHandler).Methods("GET")
	bearerTokenAuthRouter.HandleFunc("/auth/realms/redhat-external/apis/service_accounts/v1/{id}", mockServer.getServiceAccountHandler).Methods("GET")
	bearerTokenAuthRouter.HandleFunc("/auth/realms/redhat-external/apis/service_accounts/v1/{id}", mockServer.deleteServiceAccountHandler).Methods("DELETE")
	bearerTokenAuthRouter.HandleFunc("/auth/realms/redhat-external/apis/service_accounts/v1/{id}", mockServer.updateServiceAccountHandler).Methods("PATCH")
	bearerTokenAuthRouter.HandleFunc("/auth/realms/redhat-external/apis/service_accounts/v1/{id}/resetSecret", mockServer.regenerateSecretHandler).Methods("POST")
	bearerTokenAuthRouter.HandleFunc("/auth/realms/redhat-external/apis/beta/acs/v1", mockServer.createDynamicClientHandler).Methods("POST")
	bearerTokenAuthRouter.HandleFunc("/auth/realms/redhat-external/apis/beta/acs/v1/{clientId}", mockServer.deleteDynamicClientHandler).Methods("DELETE")

	mockServer.server = httptest.NewUnstartedServer(r)

	mockServer.initialClientID = "clientId"
	mockServer.initialClientSecret = "secret" // pragma: allowlist secret
	mockServer.generateAuthToken()
}

func (mockServer *redhatSSOMock) getTokenHandler(w http.ResponseWriter, r *http.Request) {
	resp := getTokenResponseMock{
		AccessToken:      mockServer.sessionAuthToken,
		ExpiresIn:        0,
		RefreshExpiresIn: 0,
		TokenType:        "Bearer",
		NotBeforePolicy:  0,
		Scope:            "profile email",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	data, _ := json.Marshal(resp)
	_, _ = w.Write(data)
}

func (mockServer *redhatSSOMock) deleteServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	if id, ok := vars["id"]; ok {
		if _, ok := mockServer.serviceAccounts[id]; ok {
			delete(mockServer.serviceAccounts, id)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

func (mockServer *redhatSSOMock) updateServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	data, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var update serviceaccountsclient.ServiceAccountRequestData
	err = json.Unmarshal(data, &update)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	updateField := func(old *string, new *string) {
		if new != nil {
			*old = *new
		}
	}

	if id, ok := vars["id"]; ok {
		if serviceAccount, ok := mockServer.serviceAccounts[id]; ok {
			updateField(serviceAccount.Name, update.Name)
			updateField(serviceAccount.Description, update.Description)

			data, err := json.Marshal(serviceAccount)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

func (mockServer *redhatSSOMock) getServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	if id, ok := vars["id"]; ok {
		if serviceAccount, ok := mockServer.serviceAccounts[id]; ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			data, _ := json.Marshal(serviceAccount)
			_, _ = w.Write(data)
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

func (mockServer *redhatSSOMock) getServiceAccountsHandler(w http.ResponseWriter, r *http.Request) {
	res := make([]serviceaccountsclient.ServiceAccountData, 0)
	for _, data := range mockServer.serviceAccounts {
		res = append(res, data)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	data, _ := json.Marshal(res)
	_, _ = w.Write(data)
}

func (mockServer *redhatSSOMock) createServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	requestData, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	if mockServer.serviceAccountsLimit != Unlimited {
		if len(mockServer.serviceAccounts) >= mockServer.serviceAccountsLimit {
			// return an error
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(fmt.Sprintf(`{ "error": "service_account_limit_exceeded", "error_description": "Max allowed number:%d of service accounts for user has reached"}`, mockServer.serviceAccountsLimit)))
			return
		}
	}

	if err != nil {
		// Ignoring real body (json)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var serviceAccountCreateRequestData serviceaccountsclient.ServiceAccountCreateRequestData
	err = json.Unmarshal(requestData, &serviceAccountCreateRequestData)
	if err != nil {
		// Ignoring real body (json)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id := uuid.New().String()
	clientID := uuid.New().String()
	secret := uuid.New().String()

	serviceAccountData := serviceaccountsclient.ServiceAccountData{
		Id:          &id,
		ClientId:    &clientID,
		Secret:      &secret,
		Name:        &serviceAccountCreateRequestData.Name,
		Description: serviceAccountCreateRequestData.Description,
	}

	mockServer.serviceAccounts[id] = serviceAccountData

	data, _ := json.Marshal(serviceAccountData)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (mockServer *redhatSSOMock) regenerateSecretHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	if id, ok := vars["id"]; ok {
		if serviceAccount, ok := mockServer.serviceAccounts[id]; ok {
			*serviceAccount.Secret = uuid.New().String()
			data, err := json.Marshal(serviceAccount)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

func (mockServer *redhatSSOMock) deleteDynamicClientHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if clientID, ok := vars["clientId"]; ok {
		if _, ok := mockServer.dynamicClients[clientID]; ok {
			delete(mockServer.dynamicClients, clientID)
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (mockServer *redhatSSOMock) createDynamicClientHandler(w http.ResponseWriter, r *http.Request) {
	requestData, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		// Ignoring real body (json)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var acsClientRequestData api.AcsClientRequestData
	err = json.Unmarshal(requestData, &acsClientRequestData)
	if err != nil {
		// Ignoring real body (json)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	clientID := uuid.New().String()
	secret := uuid.New().String()

	acsClientResponseData := api.AcsClientResponseData{
		ClientId: clientID,
		Secret:   secret, // pragma: allowlist secret
		Name:     acsClientRequestData.Name,
	}

	mockServer.dynamicClients[clientID] = acsClientResponseData

	data, _ := json.Marshal(acsClientResponseData)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// generateAuthToken ...
func (mockServer *redhatSSOMock) generateAuthToken() string {
	token := uuid.New().String()
	mockServer.authTokens = append(mockServer.authTokens, token)
	if mockServer.sessionAuthToken == "" {
		mockServer.sessionAuthToken = token
	}
	return token
}
