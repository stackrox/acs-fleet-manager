package migrate

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func NewRollbackLast(env *environments.Env) *cobra.Command {
	return &cobra.Command{
		Use:   "rollback-last",
		Short: "rollback the last migration applied",
		Long:  "rollback the last migration applied",
		Run: func(cmd *cobra.Command, args []string) {
			env.MustInvoke(func(migrations []*db.Migration) {
				glog.Infoln("Rolling back the last migration")
				for _, migration := range migrations {
					migration.RollbackLast()
					glog.Infof("Database has %d %s applied", migration.CountMigrationsApplied(), migration.GormOptions.TableName)
				}
			})
		},
	}
}
