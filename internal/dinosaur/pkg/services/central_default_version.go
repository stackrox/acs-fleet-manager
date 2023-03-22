package services

import (
	"fmt"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
)

var allowedVersionPrefixes = []string{
	"quay.io/rhacs-eng",
	"registry.redhat.io/advanced-cluster-security",
	"quay.io/stackrox-io",
}

// CentralDefaultVersionService defines methods for managing
// CentralDefaultVersion values through API and on startup of fleet-manager
type CentralDefaultVersionService interface {
	environments.BootService
	SetDefaultVersion(string) error
	GetDefaultVersion() (string, error)
}

type centralDefaultVersionService struct {
	connectionFactory *db.ConnectionFactory
	centralConfig     *config.CentralConfig
}

// Start sets the central default version in the database
// to the version string specified by the configuration
func (c *centralDefaultVersionService) Start() {
	if c.centralConfig.CentralDefaultVersion == "" {
		return
	}

	currentVersion, err := c.GetDefaultVersion()
	if err != nil {
		glog.Errorf("error getting current central default version: %s", err)
		os.Exit(1)
	}

	if currentVersion == c.centralConfig.CentralDefaultVersion {
		return
	}

	if err := c.SetDefaultVersion(c.centralConfig.CentralDefaultVersion); err != nil {
		glog.Errorf("setting central default version to: %s: %s", c.centralConfig.CentralDefaultVersion, err)
		os.Exit(1)
	}

	glog.Info("set central default version to: ", c.centralConfig.CentralDefaultVersion)

}

// Stop is a noop function implemented to satisfy environments.BootService interface
func (c *centralDefaultVersionService) Stop() {}

func (c *centralDefaultVersionService) SetDefaultVersion(version string) error {
	if !isVersionAllowed(version) {
		return fmt.Errorf("version: %s does not match one of the allowed version prefixes: %s", version, allowedVersionPrefixes)
	}

	dbConn := c.connectionFactory.New()
	versionEntry := &dbapi.CentralDefaultVersion{Version: version}

	if err := dbConn.Create(versionEntry).Error; err != nil {
		return fmt.Errorf("failed creating central default version entry: %s", err)
	}

	return nil
}

func (c *centralDefaultVersionService) GetDefaultVersion() (string, error) {
	dbConn := c.connectionFactory.New()
	defaultVersion := &dbapi.CentralDefaultVersion{}
	if err := dbConn.Order("id DESC").First(defaultVersion).Error; err != nil {
		return "", fmt.Errorf("failed getting central default version: %s", err)
	}

	return defaultVersion.Version, nil
}

func isVersionAllowed(version string) bool {
	for _, v := range allowedVersionPrefixes {
		if strings.HasPrefix(version, v) {
			return true
		}
	}

	return false
}

// NewCentralDefaultVersionService return a new instance of a CentralDefaultVersionService
func NewCentralDefaultVersionService(connectionFactory *db.ConnectionFactory, centralConfig *config.CentralConfig) CentralDefaultVersionService {
	return &centralDefaultVersionService{
		connectionFactory: connectionFactory,
		centralConfig:     centralConfig,
	}
}
