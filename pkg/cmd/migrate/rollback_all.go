package migrate

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

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
