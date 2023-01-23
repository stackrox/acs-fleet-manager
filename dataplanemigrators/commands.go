// Package dataplanemigrators contains migrations that should be run on dataplane clusters.
package dataplanemigrators

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/dataplanemigrators/incident20230120"
)

// Commands provides all commands that resolve incidents or required changes on existing dataplane clusters and/or
// central instances.
func Commands() []*cobra.Command {
	return []*cobra.Command{incident20230120.Command()}
}
