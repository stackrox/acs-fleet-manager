package gitops

import _ "embed"

//go:embed default_central.yaml
var defaultCentralTemplate []byte
