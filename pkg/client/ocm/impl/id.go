package impl

import (
	"fmt"

	"github.com/rs/xid"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
)

// MaxClusterNameLength ...
const (
	// MaxClusterNameLength - defines maximum length of an OSD cluster name
	MaxClusterNameLength = 15
)

var _ ocm.IDGenerator = idGenerator{}

// idGenerator internal implementation of IDGenerator.
type idGenerator struct {
	// prefix a string to prefix to any generated ID.
	prefix string
}

// NewIDGenerator create a new default implementation of IDGenerator.
func NewIDGenerator(prefix string) ocm.IDGenerator {
	return idGenerator{
		prefix: prefix,
	}
}

// Generate It is not allowed for the cluster name to be longer than 15 characters, hence
// the truncation
func (i idGenerator) Generate() string {
	return fmt.Sprintf("%s%s", i.prefix, xid.New().String())[0:MaxClusterNameLength]
}
