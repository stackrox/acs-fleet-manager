package server

import (
	"github.com/spf13/pflag"
)

// ServerConfig ...
type ServerConfig struct {
	BindAddress   string `json:"bind_address"`
	HTTPSCertFile string `json:"https_cert_file"`
	HTTPSKeyFile  string `json:"https_key_file"`
	EnableHTTPS   bool   `json:"enable_https"`
	// The public http host URL to access the service
	// For staging it is "https://api.stage.openshift.com"
	// For production it is "https://api.openshift.com"
	PublicHostURL         string `json:"public_url"`
	EnableTermsAcceptance bool   `json:"enable_terms_acceptance"`
	EnableLeaderElection  bool   `json:"enable_leader_election"`
}

// NewServerConfig ...
func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		BindAddress:          "localhost:8000",
		EnableHTTPS:          false,
		HTTPSCertFile:        "",
		HTTPSKeyFile:         "",
		PublicHostURL:        "http://localhost",
		EnableLeaderElection: true,
	}
}

// AddFlags ...
func (s *ServerConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.BindAddress, "api-server-bindaddress", s.BindAddress, "API server bind adddress")
	fs.StringVar(&s.HTTPSCertFile, "https-cert-file", s.HTTPSCertFile, "The path to the tls.crt file.")
	fs.StringVar(&s.HTTPSKeyFile, "https-key-file", s.HTTPSKeyFile, "The path to the tls.key file.")
	fs.BoolVar(&s.EnableHTTPS, "enable-https", s.EnableHTTPS, "Enable HTTPS rather than HTTP")
	fs.BoolVar(&s.EnableTermsAcceptance, "enable-terms-acceptance", s.EnableTermsAcceptance, "Enable terms acceptance check")
	fs.StringVar(&s.PublicHostURL, "public-host-url", s.PublicHostURL, "Public http host URL of the service")
	fs.BoolVar(&s.EnableLeaderElection, "enable-leader-election", s.EnableLeaderElection, "Enable leader election")
}

// ReadFiles ...
func (s *ServerConfig) ReadFiles() error {
	return nil
}
