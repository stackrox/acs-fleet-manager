package keycloak

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/shared"

	"github.com/spf13/pflag"
)

type KeycloakConfig struct {
	BaseURL               string               `json:"base_url"`
	Debug                 bool                 `json:"debug"`
	InsecureSkipVerify    bool                 `json:"insecure-skip-verify"`
	DinosaurRealm         *KeycloakRealmConfig `json:"dinosaur_realm"`
	OSDClusterIDPRealm    *KeycloakRealmConfig `json:"osd_cluster_idp_realm"`
	MaxLimitForGetClients int                  `json:"max_limit_for_get_clients"`
	KeycloakClientExpire  bool                 `json:"keycloak_client_expire"`
}

type KeycloakRealmConfig struct {
	Realm            string `json:"realm"`
	ClientID         string `json:"client-id"`
	ClientIDFile     string `json:"client-id_file"`
	ClientSecret     string `json:"client-secret"`
	ClientSecretFile string `json:"client-secret_file"`
	GrantType        string `json:"grant_type"`
	TokenEndpointURI string `json:"token_endpoint_uri"`
	JwksEndpointURI  string `json:"jwks_endpoint_uri"`
	ValidIssuerURI   string `json:"valid_issuer_uri"`
}

func (c *KeycloakRealmConfig) setDefaultURIs(baseURL string) {
	c.ValidIssuerURI = baseURL + "/auth/realms/" + c.Realm
	c.JwksEndpointURI = baseURL + "/auth/realms/" + c.Realm + "/protocol/openid-connect/certs"
	c.TokenEndpointURI = baseURL + "/auth/realms/" + c.Realm + "/protocol/openid-connect/token"
}

func NewKeycloakConfig() *KeycloakConfig {
	kc := &KeycloakConfig{
		DinosaurRealm: &KeycloakRealmConfig{
			ClientIDFile:     "secrets/keycloak-service.clientId",
			ClientSecretFile: "secrets/keycloak-service.clientSecret",
			GrantType:        "client_credentials",
		},
		OSDClusterIDPRealm: &KeycloakRealmConfig{
			ClientIDFile:     "secrets/osd-idp-keycloak-service.clientId",
			ClientSecretFile: "secrets/osd-idp-keycloak-service.clientSecret",
			GrantType:        "client_credentials",
		},
		Debug:                 false,
		InsecureSkipVerify:    false,
		MaxLimitForGetClients: 100,
		KeycloakClientExpire:  false,
	}
	return kc
}

func (kc *KeycloakConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&kc.DinosaurRealm.ClientIDFile, "sso-client-id-file", kc.DinosaurRealm.ClientIDFile, "File containing Keycloak privileged account client-id that has access to the Dinosaur service accounts realm")
	fs.StringVar(&kc.DinosaurRealm.ClientSecretFile, "sso-client-secret-file", kc.DinosaurRealm.ClientSecretFile, "File containing Keycloak privileged account client-secret that has access to the Dinosaur service accounts realm")
	fs.StringVar(&kc.BaseURL, "sso-base-url", kc.BaseURL, "The base URL of the sso, integration by default")
	fs.StringVar(&kc.DinosaurRealm.Realm, "sso-realm", kc.DinosaurRealm.Realm, "Realm for Dinosaur service accounts in the sso")
	fs.BoolVar(&kc.Debug, "sso-debug", kc.Debug, "Debug flag for Keycloak API")
	fs.BoolVar(&kc.InsecureSkipVerify, "sso-insecure", kc.InsecureSkipVerify, "Disable tls verification with sso")
	fs.StringVar(&kc.OSDClusterIDPRealm.ClientIDFile, "osd-idp-sso-client-id-file", kc.OSDClusterIDPRealm.ClientIDFile, "File containing Keycloak privileged account client-id that has access to the OSD Cluster IDP realm")
	fs.StringVar(&kc.OSDClusterIDPRealm.ClientSecretFile, "osd-idp-sso-client-secret-file", kc.OSDClusterIDPRealm.ClientSecretFile, "File containing Keycloak privileged account client-secret that has access to the OSD Cluster IDP realm")
	fs.StringVar(&kc.OSDClusterIDPRealm.Realm, "osd-idp-sso-realm", kc.OSDClusterIDPRealm.Realm, "Realm for OSD cluster IDP clients in the sso")
	fs.IntVar(&kc.MaxLimitForGetClients, "max-limit-for-sso-get-clients", kc.MaxLimitForGetClients, "Max limits for SSO get clients")
	fs.BoolVar(&kc.KeycloakClientExpire, "keycloak-client-expire", kc.KeycloakClientExpire, "Whether or not to tag Keycloak created Client to expire in 2 hours (useful for cleaning up after integrations tests)")
}

func (kc *KeycloakConfig) ReadFiles() error {
	err := shared.ReadFileValueString(kc.DinosaurRealm.ClientIDFile, &kc.DinosaurRealm.ClientID)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(kc.DinosaurRealm.ClientSecretFile, &kc.DinosaurRealm.ClientSecret)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(kc.OSDClusterIDPRealm.ClientIDFile, &kc.OSDClusterIDPRealm.ClientID)
	if err != nil {
		return err
	}
	err = shared.ReadFileValueString(kc.OSDClusterIDPRealm.ClientSecretFile, &kc.OSDClusterIDPRealm.ClientSecret)
	if err != nil {
		return err
	}

	kc.DinosaurRealm.setDefaultURIs(kc.BaseURL)
	kc.OSDClusterIDPRealm.setDefaultURIs(kc.BaseURL)
	return nil
}
