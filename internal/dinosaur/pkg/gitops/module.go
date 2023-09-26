package gitops

import "github.com/spf13/pflag"

// Module ...
type Module struct {
	ConfigPath string `json:"config_path"`
}

// NewModule ...
func NewModule() *Module {
	return &Module{
		ConfigPath: "",
	}
}

// AddFlags ...
func (s *Module) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.ConfigPath, "gitops-config-path", s.ConfigPath, "GitOps configuration path")
}

// ReadFiles ...
func (s *Module) ReadFiles() error {
	return nil
}
