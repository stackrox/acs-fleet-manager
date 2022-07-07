package sso

import (
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso"
	"github.com/stackrox/acs-fleet-manager/pkg/shared/utils/arrays"
)

var _ KeycloakServiceBuilderSelector = &keycloakServiceBuilderSelector{}
var _ KeycloakServiceBuilder = &keycloakServiceBuilder{}
var _ ACSKeycloakServiceBuilderConfigurator = &keycloakBuilderConfigurator{}

type KeycloakServiceBuilderSelector interface {
	ForACS() ACSKeycloakServiceBuilderConfigurator
}

type ACSKeycloakServiceBuilderConfigurator interface {
	WithConfiguration(config *iam.IAMConfig) KeycloakServiceBuilder
}

type KeycloakServiceBuilder interface {
	WithRealmConfig(realmConfig *iam.IAMRealmConfig) KeycloakServiceBuilder
	Build() IAMService
}

type keycloakServiceBuilderSelector struct {
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

type keycloakServiceBuilder struct {
	config      *iam.IAMConfig
	realmConfig *iam.IAMRealmConfig
}

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

func build(iamConfig *iam.IAMConfig, realmConfig *iam.IAMRealmConfig) IAMService {
	notNilPredicate := func(x interface{}) bool {
		return x.(*iam.IAMRealmConfig) != nil
	}

	_, newRealmConfig := arrays.FindFirst(notNilPredicate, realmConfig, iamConfig.RedhatSSORealm)
	client := redhatsso.NewSSOClient(iamConfig, newRealmConfig.(*iam.IAMRealmConfig))
	return &keycloakServiceProxy{
		accessTokenProvider: client,
		service: &redhatssoService{
			client: client,
		},
	}
}
