package db

import (
	"database/sql"
	"fmt"

	glog "github.com/stackrox/acs-fleet-manager/pkg/logging"
	mocket "github.com/selvatico/go-mocket"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	// TODO why is this imported?
	_ "github.com/lib/pq"
)

// ConnectionFactory ...
type ConnectionFactory struct {
	Config *DatabaseConfig
	DB     *gorm.DB
}

var gormConfig = &gorm.Config{
	PrepareStmt:       true,
	AllowGlobalUpdate: false, // change it to true to allow updates without the WHERE clause
	QueryFields:       true,
	Logger:            customLoggerWithMetricsCollector{},
}

// NewConnectionFactory will initialize a singleton ConnectionFactory as needed and return the same instance.
// Go includes database connection pooling in the platform. Gorm uses the same and provides a method to
// clone a connection via New(), which is safe for use by concurrent Goroutines.
func NewConnectionFactory(config *DatabaseConfig) (*ConnectionFactory, func()) {
	var db *gorm.DB
	var err error
	// refer to https://gorm.io/docs/gorm_config.html

	if config.Dialect == "postgres" {
		db, err = gorm.Open(postgres.Open(config.ConnectionString()), gormConfig)
	} else {
		// TODO what other dialects do we support?
		panic(fmt.Sprintf("Unsupported DB dialect: %s", config.Dialect))
	}
	if err != nil {
		panic(fmt.Sprintf(
			"failed to connect to %s database %s with connection string: %s\nError: %s",
			config.Dialect,
			config.Name,
			config.LogSafeConnectionString(),
			err.Error(),
		))
	}
	sqlDB, sqlDBErr := db.DB()
	if sqlDBErr != nil {
		panic(fmt.Errorf("Unexpected connection error: %s", sqlDBErr))
	}
	if err := sqlDB.Ping(); err != nil {
		panic(fmt.Errorf("unexpected connection error: %s", err))
	}

	sqlDB.SetMaxOpenConns(config.MaxOpenConnections)
	dbFactory := &ConnectionFactory{Config: config, DB: db}
	cleanup := func() {
		if err := dbFactory.close(); err != nil {
			glog.Fatalf("Unable to close db connection: %s", err.Error())
		}
	}
	return dbFactory, cleanup
}

// NewMockConnectionFactory should only be used for defining mock database drivers
// This uses mocket under the hood, use the global mocket.Catcher to change how the database should respond to SQL
// queries
func NewMockConnectionFactory(dbConfig *DatabaseConfig) *ConnectionFactory {
	if dbConfig == nil {
		dbConfig = &DatabaseConfig{}
	}
	mocket.Catcher.Register()
	mocket.Catcher.Logging = true
	sqlDB, err := sql.Open(mocket.DriverName, "connection_string")
	if err != nil {
		panic(err)
	}
	mocketDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}))
	if err != nil {
		panic(err)
	}
	connectionFactory := &ConnectionFactory{dbConfig, mocketDB}
	return connectionFactory
}

// New returns a new database connection
func (f *ConnectionFactory) New() *gorm.DB {
	if f.Config.Debug {
		return f.DB.Debug()
	}
	return f.DB
}

// CheckConnection Checks to ensure a connection is present
func (f *ConnectionFactory) CheckConnection() error {
	return f.DB.Exec("SELECT 1").Error
}

// close will close the connection to the database.
// THIS MUST **NOT** BE CALLED UNTIL THE SERVER/PROCESS IS EXITING!!
// This should only ever be called once for the entire duration of the application and only at the end.
func (f *ConnectionFactory) close() error {
	sqlDB, sqlDBErr := f.DB.DB()
	if sqlDBErr != nil {
		return fmt.Errorf("getting DB connection: %w", sqlDBErr)
	}

	err := sqlDB.Close()
	if err != nil {
		return fmt.Errorf("closing SQL DB: %w", err)
	}
	return nil
}

// By default do no roll back transaction.
// only perform rollback if explicitly set by db.db.MarkForRollback(ctx, err)
const defaultRollbackPolicy = false

// TxFactory represents an sql transaction
type txFactory struct {
	resolved          bool
	rollbackFlag      bool
	tx                *sql.Tx
	txid              int64
	postCommitActions []func()
	db                *sql.DB
}

// newTransaction constructs a new Transaction object.
func (f *ConnectionFactory) newTransaction() (*txFactory, error) {
	sqlDB, sqlDBErr := f.DB.DB()
	if sqlDBErr != nil {
		return nil, fmt.Errorf("getting DB connection: %w", sqlDBErr)
	}
	tx := &txFactory{
		db: sqlDB,
	}
	return tx, tx.begin()
}

func (f *txFactory) begin() error {
	tx, err := f.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning DB transaction: %w", err)
	}

	var txid int64

	// current transaction ID set by postgres.  these are *not* distinct across time
	// and do get reset after postgres performs "vacuuming" to reclaim used IDs.
	row := tx.QueryRow("select txid_current()")
	if row != nil {
		err := row.Scan(&txid)
		if err != nil {
			return fmt.Errorf("scanning row: %w", err)
		}
	}

	f.tx = tx
	f.txid = txid
	f.resolved = false
	f.rollbackFlag = defaultRollbackPolicy
	return nil
}

// markedForRollback returns true if a transaction is flagged for rollback and false otherwise.
func (f *txFactory) markedForRollback() bool {
	return f.rollbackFlag
}
