package sso

import (
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso"
	"github.com/stackrox/acs-fleet-manager/pkg/shared/utils/arrays"
)

var _ KeycloakServiceBuilderSelector = &keycloakServiceBuilderSelector{}
var _ KeycloakServiceBuilder = &keycloakServiceBuilder{}
var _ ACSKeycloakServiceBuilderConfigurator = &keycloakBuilderConfigurator{}
var _ OSDKeycloakServiceBuilderConfigurator = &osdBuilderConfigurator{}

type KeycloakServiceBuilderSelector interface {
	ForOSD() OSDKeycloakServiceBuilderConfigurator
	ForACS() ACSKeycloakServiceBuilderConfigurator
}

type ACSKeycloakServiceBuilderConfigurator interface {
	WithConfiguration(config *iam.IAMConfig) KeycloakServiceBuilder
}

type OSDKeycloakServiceBuilderConfigurator interface {
	WithConfiguration(config *iam.IAMConfig) OSDKeycloakServiceBuilder
}

type KeycloakServiceBuilder interface {
	WithRealmConfig(realmConfig *iam.IAMRealmConfig) KeycloakServiceBuilder
	Build() IAMService
}

type OSDKeycloakServiceBuilder interface {
	WithRealmConfig(realmConfig *iam.IAMRealmConfig) OSDKeycloakServiceBuilder
	Build() OSDKeycloakService
}

type keycloakServiceBuilderSelector struct {
}

func (s *keycloakServiceBuilderSelector) ForOSD() OSDKeycloakServiceBuilderConfigurator {
	return &osdBuilderConfigurator{}
}

func (s *keycloakServiceBuilderSelector) ForACS() ACSKeycloakServiceBuilderConfigurator {
	return &keycloakBuilderConfigurator{}
}

type keycloakBuilderConfigurator struct{}
type osdBuilderConfigurator keycloakBuilderConfigurator

func (k *keycloakBuilderConfigurator) WithConfiguration(config *iam.IAMConfig) KeycloakServiceBuilder {
	return &keycloakServiceBuilder{
		config: config,
	}
}

func (o *osdBuilderConfigurator) WithConfiguration(config *iam.IAMConfig) OSDKeycloakServiceBuilder {
	return &osdKeycloackServiceBuilder{
		config: config,
	}
}

type keycloakServiceBuilder struct {
	config      *iam.IAMConfig
	realmConfig *iam.IAMRealmConfig
}

type osdKeycloackServiceBuilder keycloakServiceBuilder

// Build returns an instance of IAMService ready to be used.
// If a custom realm is configured (WithRealmConfig called), then always Keycloak provider is used
// irrespective of the `builder.config.SelectSSOProvider` value
func (builder *keycloakServiceBuilder) Build() IAMService {
	return build(builder.config, builder.realmConfig)
}

func (builder *keycloakServiceBuilder) WithRealmConfig(realmConfig *iam.IAMRealmConfig) KeycloakServiceBuilder {
	builder.realmConfig = realmConfig
	return builder
}

// Build returns an instance of IAMService ready to be used.
// If a custom realm is configured (WithRealmConfig called), then always Keycloak provider is used
// irrespective of the `builder.config.SelectSSOProvider` value
func (builder *osdKeycloackServiceBuilder) Build() OSDKeycloakService {
	return build(builder.config, builder.realmConfig).(OSDKeycloakService)
}

func (builder *osdKeycloackServiceBuilder) WithRealmConfig(realmConfig *iam.IAMRealmConfig) OSDKeycloakServiceBuilder {
	builder.realmConfig = realmConfig
	return builder
}

func build(keycloakConfig *iam.IAMConfig, realmConfig *iam.IAMRealmConfig) IAMService {
	notNilPredicate := func(x interface{}) bool {
		return x.(*iam.IAMRealmConfig) != nil
	}

	_, newRealmConfig := arrays.FindFirst(notNilPredicate, realmConfig, keycloakConfig.RedhatSSORealm)
	client := redhatsso.NewSSOClient(keycloakConfig, newRealmConfig.(*iam.IAMRealmConfig))
	return &keycloakServiceProxy{
		accessTokenProvider: client,
		service: &redhatssoService{
			client: client,
		},
	}
}
