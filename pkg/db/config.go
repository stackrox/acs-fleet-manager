package db

import (
	"fmt"

	"github.com/stackrox/acs-fleet-manager/pkg/shared"

	"github.com/spf13/pflag"
)

// DatabaseConfig ...
type DatabaseConfig struct {
	Dialect            string `json:"dialect"`
	SSLMode            string `json:"sslmode"`
	Debug              bool   `json:"debug"`
	MaxOpenConnections int    `json:"max_connections"`

	Host     string `json:"host"`
	Port     int    `json:"port"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`

	DatabaseCaCertFile string `json:"db_ca_cert_file"`
	HostFile           string `json:"host_file"`
	PortFile           string `json:"port_file"`
	NameFile           string `json:"name_file"`
	UsernameFile       string `json:"username_file"`
	PasswordFile       string `json:"password_file"`
}

// NewDatabaseConfig ...
func NewDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Dialect:            "postgres",
		SSLMode:            "disable",
		Debug:              false,
		MaxOpenConnections: 50,

		HostFile:           "secrets/db.host",
		PortFile:           "secrets/db.port",
		UsernameFile:       "secrets/db.user",
		PasswordFile:       "secrets/db.password",
		NameFile:           "secrets/db.name",
		DatabaseCaCertFile: "secrets/db.ca_cert",
	}
}

// AddFlags ...
func (c *DatabaseConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.DatabaseCaCertFile, "db-ssl-certificate-file", c.DatabaseCaCertFile, "Database ssl cert string file")
	fs.StringVar(&c.HostFile, "db-host-file", c.HostFile, "Database host string file")
	fs.StringVar(&c.PortFile, "db-port-file", c.PortFile, "Database port file")
	fs.StringVar(&c.UsernameFile, "db-user-file", c.UsernameFile, "Database username file")
	fs.StringVar(&c.PasswordFile, "db-password-file", c.PasswordFile, "Database password file")
	fs.StringVar(&c.NameFile, "db-name-file", c.NameFile, "Database name file")
	fs.StringVar(&c.SSLMode, "db-sslmode", c.SSLMode, "Database ssl mode (disable | require | verify-ca | verify-full)")
	fs.BoolVar(&c.Debug, "enable-db-debug", c.Debug, " framework's debug mode")
	fs.IntVar(&c.MaxOpenConnections, "db-max-open-connections", c.MaxOpenConnections, "Maximum open DB connections for this instance")
}

// ReadFiles ...
func (c *DatabaseConfig) ReadFiles() error {

	err := shared.ReadFileValueString(c.HostFile, &c.Host)
	if err != nil {
		return err
	}

	err = shared.ReadFileValueInt(c.PortFile, &c.Port)
	if err != nil {
		return err
	}

	err = shared.ReadFileValueString(c.UsernameFile, &c.Username)
	if err != nil {
		return err
	}

	err = shared.ReadFileValueString(c.PasswordFile, &c.Password)
	if err != nil {
		return err
	}

	err = shared.ReadFileValueString(c.NameFile, &c.Name)
	return err
}

// ConnectionString ...
func (c *DatabaseConfig) ConnectionString() string {
	if c.SSLMode != "disable" {
		return fmt.Sprintf(
			"host=%s port=%d user=%s password='%s' dbname=%s sslmode=%s sslrootcert=%s",
			c.Host, c.Port, c.Username, c.Password, c.Name, c.SSLMode, c.DatabaseCaCertFile,
		)
	}
	return fmt.Sprintf(
		"host=%s port=%d user=%s password='%s' dbname=%s sslmode=%s",
		c.Host, c.Port, c.Username, c.Password, c.Name, c.SSLMode,
	)
}

// LogSafeConnectionString ...
func (c *DatabaseConfig) LogSafeConnectionString() string {
	if c.SSLMode != "disable" {
		return fmt.Sprintf(
			"host=%s port=%d user=%s password='<REDACTED>' dbname=%s sslmode=%s sslrootcert=<REDACTED>",
			c.Host, c.Port, c.Username, c.Name, c.SSLMode,
		)
	}
	return fmt.Sprintf(
		"host=%s port=%d user=%s password='<REDACTED>' dbname=%s sslmode=%s",
		c.Host, c.Port, c.Username, c.Name, c.SSLMode,
	)
}
