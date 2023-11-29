package shared

import (
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/stackrox/rox/pkg/maputil"
)

// AddonParameters addon parameters
type AddonParameters map[string]string

// Addon contains information about the addons on the cluster
type Addon struct {
	ID           string
	Version      string
	SourceImage  string
	PackageImage string
	Parameters   AddonParameters
}

// SHA256Sum returns SHA256 checksum of the addon parameters
func (p AddonParameters) SHA256Sum() string {
	keys := maputil.Keys(p)
	sort.Strings(keys) // to make hash generation deterministic

	h := sha256.New()

	for _, k := range keys {
		v := p[k]
		b := sha256.Sum256([]byte(k))
		h.Write(b[:])
		b = sha256.Sum256([]byte(v))
		h.Write(b[:])
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
