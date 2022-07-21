package migrate

import (
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
)

// NewRollbackAll ...
func NewRollbackAll(env *environments.Env) *cobra.Command {
	return &cobra.Command{
		Use:   "rollback-all",
		Short: "rollback all migrations",
		Long:  "rollback all migrations",
		Run: func(cmd *cobra.Command, args []string) {
			env.MustInvoke(func(migrations []*db.Migration) {
				glog.Infoln("Rolling back all applied migrations")
				for _, migration := range migrations {
					migration.RollbackAll()
					glog.Infof("Database has %d %s applied", migration.CountMigrationsApplied(), migration.GormOptions.TableName)
				}
			})
		},
	}
}
