package sso

import (
	"github.com/stackrox/acs-fleet-manager/pkg/client/keycloak"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso"
	"github.com/stackrox/acs-fleet-manager/pkg/shared/utils/arrays"
)

var _ KeycloakServiceBuilderSelector = &keycloakServiceBuilderSelector{}
var _ KeycloakServiceBuilder = &keycloakServiceBuilder{}
var _ KFMKeycloakServiceBuilderConfigurator = &keycloakBuilderConfigurator{}
var _ OSDKeycloakServiceBuilderConfigurator = &osdBuilderConfigurator{}

type KeycloakServiceBuilderSelector interface {
	ForOSD() OSDKeycloakServiceBuilderConfigurator
	ForACS() KFMKeycloakServiceBuilderConfigurator
}

type KFMKeycloakServiceBuilderConfigurator interface {
	WithConfiguration(config *keycloak.KeycloakConfig) KeycloakServiceBuilder
}

type OSDKeycloakServiceBuilderConfigurator interface {
	WithConfiguration(config *keycloak.KeycloakConfig) OSDKeycloakServiceBuilder
}

type KeycloakServiceBuilder interface {
	WithRealmConfig(realmConfig *keycloak.KeycloakRealmConfig) KeycloakServiceBuilder
	Build() KeycloakService
}

type OSDKeycloakServiceBuilder interface {
	WithRealmConfig(realmConfig *keycloak.KeycloakRealmConfig) OSDKeycloakServiceBuilder
	Build() OSDKeycloakService
}

type keycloakServiceBuilderSelector struct {
}

func (s *keycloakServiceBuilderSelector) ForOSD() OSDKeycloakServiceBuilderConfigurator {
	return &osdBuilderConfigurator{}
}

func (s *keycloakServiceBuilderSelector) ForACS() KFMKeycloakServiceBuilderConfigurator {
	return &keycloakBuilderConfigurator{}
}

type keycloakBuilderConfigurator struct{}
type osdBuilderConfigurator keycloakBuilderConfigurator

func (k *keycloakBuilderConfigurator) WithConfiguration(config *keycloak.KeycloakConfig) KeycloakServiceBuilder {
	return &keycloakServiceBuilder{
		config: config,
	}
}

func (o *osdBuilderConfigurator) WithConfiguration(config *keycloak.KeycloakConfig) OSDKeycloakServiceBuilder {
	return &osdKeycloackServiceBuilder{
		config: config,
	}
}

type keycloakServiceBuilder struct {
	config      *keycloak.KeycloakConfig
	realmConfig *keycloak.KeycloakRealmConfig
}

type osdKeycloackServiceBuilder keycloakServiceBuilder

// Build returns an instance of KeycloakService ready to be used.
// If a custom realm is configured (WithRealmConfig called), then always Keycloak provider is used
// irrespective of the `builder.config.SelectSSOProvider` value
func (builder *keycloakServiceBuilder) Build() KeycloakService {
	return build(builder.config, builder.realmConfig)
}

func (builder *keycloakServiceBuilder) WithRealmConfig(realmConfig *keycloak.KeycloakRealmConfig) KeycloakServiceBuilder {
	builder.realmConfig = realmConfig
	return builder
}

// Build returns an instance of KeycloakService ready to be used.
// If a custom realm is configured (WithRealmConfig called), then always Keycloak provider is used
// irrespective of the `builder.config.SelectSSOProvider` value
func (builder *osdKeycloackServiceBuilder) Build() OSDKeycloakService {
	return build(builder.config, builder.realmConfig).(OSDKeycloakService)
}

func (builder *osdKeycloackServiceBuilder) WithRealmConfig(realmConfig *keycloak.KeycloakRealmConfig) OSDKeycloakServiceBuilder {
	builder.realmConfig = realmConfig
	return builder
}

func build(keycloakConfig *keycloak.KeycloakConfig, realmConfig *keycloak.KeycloakRealmConfig) KeycloakService {
	notNilPredicate := func(x interface{}) bool {
		return x.(*keycloak.KeycloakRealmConfig) != nil
	}

	_, newRealmConfig := arrays.FindFirst(notNilPredicate, realmConfig, keycloakConfig.RedhatSSORealm)
	client := redhatsso.NewSSOClient(keycloakConfig, newRealmConfig.(*keycloak.KeycloakRealmConfig))
	return &keycloakServiceProxy{
		accessTokenProvider: client,
		service: &redhatssoService{
			client: client,
		},
	}
}
