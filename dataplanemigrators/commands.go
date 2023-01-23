package dataplanemigrators

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/dataplanemigrators/incident20230120"
)

func Commands() []*cobra.Command {
	return []*cobra.Command{incident20230120.Command()}
}
