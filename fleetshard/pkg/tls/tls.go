// Package tls - temporary package to load self-signed certificates.
// TODO(ROX-11523): To be replaced by loading certs from the fleet-manager
package tls

import "embed"

//go:embed ca.crt tls.crt tls.key
var fs embed.FS

func SelfSignedCA() string {
	data, _ := fs.ReadFile("ca.crt")
	return string(data)
}

func SelfSignedCert() string {
	data, _ := fs.ReadFile("tls.crt")
	return string(data)
}

func SelfSignedKey() string {
	data, _ := fs.ReadFile("tls.key")
	return string(data)
}
