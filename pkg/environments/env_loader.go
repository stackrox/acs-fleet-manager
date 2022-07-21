package environments

// EnvLoader ...
type EnvLoader interface {
	Defaults() map[string]string
	ModifyConfiguration(env *Env) error
}

// SimpleEnvLoader ...
type SimpleEnvLoader map[string]string

var _ EnvLoader = SimpleEnvLoader{}

// Defaults ...
func (b SimpleEnvLoader) Defaults() map[string]string {
	return b
}

// ModifyConfiguration ...
func (b SimpleEnvLoader) ModifyConfiguration(env *Env) error {
	return nil
}
