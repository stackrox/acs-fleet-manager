package sso

import (
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso"
	"github.com/stackrox/acs-fleet-manager/pkg/shared/utils/arrays"
)

var _ IAMServiceBuilderSelector = &iamServiceBuilderSelector{}
var _ IAMServiceBuilder = &iamServiceBuilder{}
var _ ACSIAMServiceBuilderConfigurator = &iamBuilderConfigurator{}

type IAMServiceBuilderSelector interface {
	ForACS() ACSIAMServiceBuilderConfigurator
}

type ACSIAMServiceBuilderConfigurator interface {
	WithConfiguration(config *iam.IAMConfig) IAMServiceBuilder
}

type IAMServiceBuilder interface {
	WithRealmConfig(realmConfig *iam.IAMRealmConfig) IAMServiceBuilder
	Build() IAMService
}

type iamServiceBuilderSelector struct {
}

func (s *iamServiceBuilderSelector) ForACS() ACSIAMServiceBuilderConfigurator {
	return &iamBuilderConfigurator{}
}

type iamBuilderConfigurator struct{}

func (k *iamBuilderConfigurator) WithConfiguration(config *iam.IAMConfig) IAMServiceBuilder {
	return &iamServiceBuilder{
		config: config,
	}
}

type iamServiceBuilder struct {
	config      *iam.IAMConfig
	realmConfig *iam.IAMRealmConfig
}

// Build returns an instance of IAMService ready to be used.
// If a custom realm is configured (WithRealmConfig called), then always Keycloak provider is used
// irrespective of the `builder.config.SelectSSOProvider` value
func (builder *iamServiceBuilder) Build() IAMService {
	return build(builder.config, builder.realmConfig)
}

func (builder *iamServiceBuilder) WithRealmConfig(realmConfig *iam.IAMRealmConfig) IAMServiceBuilder {
	builder.realmConfig = realmConfig
	return builder
}

func build(iamConfig *iam.IAMConfig, realmConfig *iam.IAMRealmConfig) IAMService {
	notNilPredicate := func(x interface{}) bool {
		return x.(*iam.IAMRealmConfig) != nil
	}

	_, newRealmConfig := arrays.FindFirst(notNilPredicate, realmConfig, iamConfig.RedhatSSORealm)
	client := redhatsso.NewSSOClient(iamConfig, newRealmConfig.(*iam.IAMRealmConfig))
	return &redhatssoService{
		client: client,
	}
}
