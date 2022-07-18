package mocks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	serviceaccountsclient "github.com/redhat-developer/app-services-sdk-go/serviceaccounts/apiv1internal/client"
)

const (
	Unlimited = -1
)

type RedhatSSOMock interface {
	Start()
	Stop()
	BaseURL() string
	GenerateNewAuthToken() string
	SetBearerToken(token string) string
	DeleteAllServiceAccounts()
	ServiceAccountsLimit() int
}

type redhatSSOMock struct {
	server               *httptest.Server
	authTokens           []string
	serviceAccounts      map[string]serviceaccountsclient.ServiceAccountData
	sessionAuthToken     string
	serviceAccountsLimit int
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

type MockServerOption func(mock *redhatSSOMock)

func WithServiceAccountLimit(limit int) MockServerOption {
	return func(mock *redhatSSOMock) {
		mock.serviceAccountsLimit = limit
	}
}

func NewMockServer(options ...MockServerOption) RedhatSSOMock {
	mockServer := &redhatSSOMock{
		serviceAccounts:      make(map[string]serviceaccountsclient.ServiceAccountData),
		serviceAccountsLimit: Unlimited,
	}

	for _, option := range options {
		option(mockServer)
	}

	mockServer.init()
	return mockServer
}

func (mockServer *redhatSSOMock) ServiceAccountsLimit() int {
	return mockServer.serviceAccountsLimit
}

func (mockServer *redhatSSOMock) DeleteAllServiceAccounts() {
	mockServer.serviceAccounts = make(map[string]serviceaccountsclient.ServiceAccountData)
}

func (mockServer *redhatSSOMock) Start() {
	mockServer.server.Start()
}

func (mockServer *redhatSSOMock) Stop() {
	mockServer.server.Close()
}

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
		clientId := request.FormValue("client_id")
		clientSecret := request.FormValue("client_secret")

		if serviceAccount, ok := mockServer.serviceAccounts[clientId]; ok {
			if *serviceAccount.Secret == clientSecret {
				next.ServeHTTP(writer, request)
				return
			}
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
	bearerTokenAuthRouter.HandleFunc("/auth/realms/redhat-external/apis/service_accounts/v1/{clientId}", mockServer.getServiceAccountHandler).Methods("GET")
	bearerTokenAuthRouter.HandleFunc("/auth/realms/redhat-external/apis/service_accounts/v1/{clientId}", mockServer.deleteServiceAccountHandler).Methods("DELETE")
	bearerTokenAuthRouter.HandleFunc("/auth/realms/redhat-external/apis/service_accounts/v1/{clientId}", mockServer.updateServiceAccountHandler).Methods("PATCH")
	bearerTokenAuthRouter.HandleFunc("/auth/realms/redhat-external/apis/service_accounts/v1/{clientId}/resetSecret", mockServer.regenerateSecretHandler).Methods("POST")

	mockServer.server = httptest.NewUnstartedServer(r)
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

	if clientId, ok := vars["clientId"]; ok {
		if _, ok := mockServer.serviceAccounts[clientId]; ok {
			delete(mockServer.serviceAccounts, clientId)
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

	if clientId, ok := vars["clientId"]; ok {
		if serviceAccount, ok := mockServer.serviceAccounts[clientId]; ok {
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

	if clientId, ok := vars["clientId"]; ok {
		if serviceAccount, ok := mockServer.serviceAccounts[clientId]; ok {
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
	clientId := uuid.New().String()
	secret := uuid.New().String()

	serviceAccountData := serviceaccountsclient.ServiceAccountData{
		Id:          &id,
		ClientId:    &clientId,
		Secret:      &secret,
		Name:        &serviceAccountCreateRequestData.Name,
		Description: &serviceAccountCreateRequestData.Description,
	}

	mockServer.serviceAccounts[clientId] = serviceAccountData

	data, _ := json.Marshal(serviceAccountData)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (mockServer *redhatSSOMock) regenerateSecretHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	if clientId, ok := vars["clientId"]; ok {
		if serviceAccount, ok := mockServer.serviceAccounts[clientId]; ok {
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

func (mockServer *redhatSSOMock) GenerateNewAuthToken() string {
	token := uuid.New().String()
	mockServer.authTokens = append(mockServer.authTokens, token)
	return token
}

func (mockServer *redhatSSOMock) SetBearerToken(token string) string {
	mockServer.authTokens = append(mockServer.authTokens, token)
	mockServer.sessionAuthToken = token
	return token
}
