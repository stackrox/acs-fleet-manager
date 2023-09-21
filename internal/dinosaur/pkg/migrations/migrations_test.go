package migrations

import (
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrations(t *testing.T) {
	config := db.NewDatabaseConfig()

	migration, cleanup, err := New(config)
	require.NoError(t, err)
	defer cleanup()
	migration.Migrate()
	assert.Equal(t, 22, migration.CountMigrationsApplied())
	migration.RollbackAll()
	assert.Equal(t, 0, migration.CountMigrationsApplied())
	tx := migration.DbFactory.DB
	central := tx.Model(&private.CentralRequest{})
	require.NotNil(t, central)
	require.NotNil(t, central.Statement)
	// require.NotNil(t, central.Statement.Schema)
	assert.Equal(t, "", central.Statement.Name())
}
