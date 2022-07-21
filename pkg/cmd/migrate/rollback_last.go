package migrate

import (
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
)

// NewRollbackLast ...
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
