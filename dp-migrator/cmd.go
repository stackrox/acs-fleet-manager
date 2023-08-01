// Package dpmigrator handles data plane migrations.
package dpmigrator

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/dp-migrator/auth"
)

// Command provides the command to migrate things on the dataplane cluster.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "All migrations for data plane clusters",
	}

	cmd.AddCommand(auth.MigrateCommand())

	return cmd
}
