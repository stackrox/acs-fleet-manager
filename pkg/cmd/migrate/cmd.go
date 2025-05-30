// Package migrate ...
package migrate

import (
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"

	"github.com/stackrox/acs-fleet-manager/pkg/db"
)

// NewMigrateCommand migrate sub-command handles running migrations
func NewMigrateCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run fleet-manager data migrations",
		Long:  "Run ACS Cloud Services Fleet Manager data migrations",
		Run: func(cmd *cobra.Command, args []string) {
			env.MustInvoke(func(migrations []*db.Migration) {
				glog.Infoln("Migration starting")
				for _, migration := range migrations {
					migration.Migrate()
					glog.Infof("Database has %d %s applied", migration.CountMigrationsApplied(), migration.GormOptions.TableName)
				}
			})
		},
	}
	cmd.AddCommand(
		NewRollbackAll(env),
		NewRollbackLast(env),
	)
	return cmd
}
