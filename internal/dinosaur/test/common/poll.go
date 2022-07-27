package common

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/util/wait"
)

var defaultShouldLogFunction ShouldLogFunc = func(retry int, maxRetry int, maxRetryLogs int) bool {
	q := maxRetry / maxRetryLogs
	if q < 1 {
		q = 1
	}
	return retry == 1 || retry == maxRetry || retry%q == 0
}
var defaultLogFunction = func(pattern string, args ...interface{}) {
	p := fmt.Sprintf("%s\n", strings.TrimSuffix(pattern, "\n"))
	fmt.Printf(p, args...)
}

const (
	defaultPollInterval        = 1 * time.Second
	defaultDinosaurPollTimeout = 20 * time.Minute
	// defaultMaxRetryLogs - Maximum number of retry log to show. The waiting period between each of the logs is
	// calculated so that at maximum `defaultMaxRetryLogs` statements are logged if reaching the last attempt
	defaultMaxRetryLogs = 50

	// defaultLogMessage - The default log message is no custom log message is provided
	defaultLogMessage = "Retrying..."

	// defaultLogEnabled - sets weather by default the polling logs are enabled ot not
	defaultLogEnabled = true
)

// ShouldLogFunc ...
type ShouldLogFunc func(attempt int, maxRetries int, maxRetryLogs int) bool

// OnStartFunc ...
type OnStartFunc func(maxRetries int) error

// OnRetryFunc ...
type OnRetryFunc func(attempt int, maxRetries int) (done bool, err error)

// OnFinishFunc ...
type OnFinishFunc func(attempt int, maxRetries int, err error)

// Poller - This is the actual poller
type Poller interface {
	// Poll - starts the polling
	Poll() error
}

type dbDumper struct {
	columns []string
	filter  string
}

type poller struct {
	attempts        int
	interval        time.Duration
	onStart         OnStartFunc
	onRetry         OnRetryFunc
	onFinish        OnFinishFunc
	logEnabled      bool
	retryLogMessage string
	customLog       func(retry int, maxRetry int) string
	outputFunction  func(pattern string, args ...interface{})
	maxRetryLogs    int
	shouldLog       ShouldLogFunc

	db     *gorm.DB
	dbDump map[string]dbDumper
}

var _ Poller = &poller{}

// Poll ...
func (poller *poller) Poll() error {
	maxAttempts := poller.attempts
	start := time.Now()
	if poller.onStart != nil {
		if err := poller.onStart(maxAttempts); err != nil {
			return err
		}
	}

	attempt := 0
	errs := 0
	maxErrors := 10

	err := wait.PollImmediate(poller.interval, poller.interval*time.Duration(poller.attempts), func() (done bool, err error) {
		attempt++
		poller.logRetry(attempt, maxAttempts, start)
		finished, e := poller.onRetry(attempt, maxAttempts)
		if e != nil {
			errs++
			if errs >= maxErrors {
				return finished, e
			}
			poller.outputFunction("Error ocurred when polling (will be ignored): %+v", e)
		}
		return finished, nil
	})

	if poller.onFinish != nil {
		poller.onFinish(attempt, maxAttempts, err)
	}

	if poller.logEnabled {
		elapsed := time.Since(start).Round(time.Second)
		poller.outputFunction("%d/%d [%s] - Polling finished", attempt, maxAttempts, elapsed)
	}

	if err != nil {
		return fmt.Errorf("polling: %w", err)
	}
	return nil
}

func (poller *poller) dumpDB() {
	// database dump
	for key, value := range poller.dbDump {
		poller.outputFunction("DUMPING '%s' with filter '%s'", key, value.filter)
		// query the resource
		var results []map[string]interface{}
		if err := poller.db.Table(key).Find(&results, value.filter).Error; err != nil {
			poller.outputFunction("Error dumping the database: %+v - %s", err, key)
		} else {
			if len(results) > 0 {
				var columns []string
				if value.columns != nil && len(value.columns) > 0 {
					columns = value.columns
				} else {
					columns = make([]string, 0, len(results[0]))
					for k := range results[0] {
						columns = append(columns, k)
					}
				}
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader(columns)

				for _, v := range results {
					row := make([]string, len(columns))
					for i, col := range columns {
						row[i] = fmt.Sprintf("%v", v[col])
					}
					table.Append(row)
				}

				table.Render()
			} else {
				poller.outputFunction("{{ EMPTY }}")
			}
		}
	}
}

func (poller *poller) logRetry(attempt int, maxAttempts int, start time.Time) {
	// log every maxAttempts/maxRetryLogs attempts and log first and last attempt
	if poller.logEnabled && poller.shouldLog(attempt, maxAttempts, poller.maxRetryLogs) {
		elapsed := time.Since(start).Round(time.Second)

		poller.dumpDB()
		var msg string

		if poller.customLog != nil {
			msg = poller.customLog(attempt, maxAttempts)
		} else {
			msg = poller.retryLogMessage
		}

		poller.outputFunction("%d/%d [%s] - %s", attempt, maxAttempts, elapsed, msg)
	}
}

// LogFunctionType ...
type LogFunctionType func(pattern string, args ...interface{})

// PollerBuilder is to be used to create a poller to check for an event to happens
type PollerBuilder interface {
	// OnStart - The passed in function is executed before starting the polling. Useful for setting things up or logging the start of the polling
	OnStart(onStart OnStartFunc) PollerBuilder
	// OnRetry - The passed in function is executed at each retry. This function should contain the desired logic.
	OnRetry(onRetry OnRetryFunc) PollerBuilder
	// OnFinish - The passed in function is executed after the polling has ended.
	// Polling can end if the desired event has occurred, if the `OnRetry` function has failed or if the maximum number of retries has been reached
	OnFinish(onFinish OnFinishFunc) PollerBuilder
	// IntervalAndTimeout - Sets the interval between each retry and the timeout after which the polling will end
	// This method can be used as alternative of IntervalAndRetries
	IntervalAndTimeout(interval time.Duration, timeout time.Duration) PollerBuilder
	// IntervalAndRetries - Sets the interval between each retry and the maximum number of attempt to perform before giving up
	// This method can be used as alternative of IntervalAndTimeout
	IntervalAndRetries(interval time.Duration, maxRetries int) PollerBuilder
	// DisableRetryLog - Call this if you don't want the poller to log the polling
	DisableRetryLog() PollerBuilder
	// RetryLogMessage - Message to be shown on each retry. For more advanced logging, use RetryLogFunction
	RetryLogMessage(msg string) PollerBuilder
	// RetryLogMessagef - Message to be shown on each retry. The message is formatted according to a format specifie.
	// For more advanced logging, use RetryLogFunction
	RetryLogMessagef(format string, params ...interface{}) PollerBuilder
	// RetryLogFunction - The function to be called each time the poller desires to show some log. Default: 'Retrying...'
	RetryLogFunction(logFunction func(retry int, maxRetry int) string) PollerBuilder
	// OutputFunction - Useful only if RetryLogFunction is not used. Sets the function the poller must use to
	// output the log. Defaults to fmt.Printf
	OutputFunction(logFunction LogFunctionType) PollerBuilder
	// MaxRetryLogs - Sets the maximum number of log statements we want to be showed. They are evenly divided by the
	// maximum number of attempts
	MaxRetryLogs(maxRetryLogs int) PollerBuilder
	// ShouldLog - To be used to customise when a log should be shown or not. The default function tries to respect the
	// MaxRetryLogs value.
	ShouldLog(shouldLog ShouldLogFunc) PollerBuilder
	DumpCluster(id string) PollerBuilder
	DumpDB(name string, filter string, columns ...string) PollerBuilder
	// Build - Builds the poller
	Build() Poller
}

type pollerBuilder struct {
	p poller
}

var _ PollerBuilder = &pollerBuilder{}

// NewPollerBuilder ...
func NewPollerBuilder(db *db.ConnectionFactory) PollerBuilder {
	return &pollerBuilder{
		p: poller{
			logEnabled:      defaultLogEnabled,
			retryLogMessage: defaultLogMessage,
			maxRetryLogs:    defaultMaxRetryLogs,
			outputFunction:  defaultLogFunction,
			shouldLog:       defaultShouldLogFunction,
			db:              db.New(),
		},
	}
}

// OnStart ...
func (b *pollerBuilder) OnStart(onStart OnStartFunc) PollerBuilder {
	b.p.onStart = onStart
	return b
}

// OnRetry ...
func (b *pollerBuilder) OnRetry(onRetry OnRetryFunc) PollerBuilder {
	b.p.onRetry = onRetry
	return b
}

// OnFinish ...
func (b *pollerBuilder) OnFinish(onFinish OnFinishFunc) PollerBuilder {
	b.p.onFinish = onFinish
	return b
}

// IntervalAndRetries ...
func (b *pollerBuilder) IntervalAndRetries(interval time.Duration, maxRetries int) PollerBuilder {
	b.p.interval = interval
	b.p.attempts = maxRetries
	return b
}

// IntervalAndTimeout ...
func (b *pollerBuilder) IntervalAndTimeout(interval time.Duration, timeout time.Duration) PollerBuilder {
	b.p.interval = interval
	b.p.attempts = int((timeout-1)/interval + 1)
	return b
}

// DisableRetryLog ...
func (b *pollerBuilder) DisableRetryLog() PollerBuilder {
	b.p.logEnabled = false
	return b
}

// RetryLogMessagef ...
func (b *pollerBuilder) RetryLogMessagef(format string, params ...interface{}) PollerBuilder {
	b.p.retryLogMessage = fmt.Sprintf(format, params...)
	return b
}

// RetryLogMessage ...
func (b *pollerBuilder) RetryLogMessage(msg string) PollerBuilder {
	b.p.retryLogMessage = msg
	return b
}

// RetryLogFunction ...
func (b *pollerBuilder) RetryLogFunction(logFunction func(retry int, maxRetry int) string) PollerBuilder {
	b.p.customLog = logFunction
	return b
}

// OutputFunction ...
func (b *pollerBuilder) OutputFunction(logFunction LogFunctionType) PollerBuilder {
	b.p.outputFunction = logFunction
	return b
}

// MaxRetryLogs ...
func (b *pollerBuilder) MaxRetryLogs(maxRetryLogs int) PollerBuilder {
	b.p.maxRetryLogs = maxRetryLogs
	return b
}

// ShouldLog ...
func (b *pollerBuilder) ShouldLog(shouldLog ShouldLogFunc) PollerBuilder {
	b.p.shouldLog = shouldLog
	return b
}

// DumpCluster ...
func (b *pollerBuilder) DumpCluster(id string) PollerBuilder {
	return b.DumpDB("clusters", fmt.Sprintf("cluster_id = '%s'", id), "cluster_id", "status", "updated_at")
}

// DumpDB ...
func (b *pollerBuilder) DumpDB(name string, filter string, columns ...string) PollerBuilder {
	if b.p.dbDump == nil {
		b.p.dbDump = make(map[string]dbDumper)
	}
	b.p.dbDump[name] = dbDumper{
		filter:  filter,
		columns: columns,
	}
	return b
}

// Build ...
func (b *pollerBuilder) Build() Poller {
	if b.p.interval == 0 {
		panic("no retry interval has been specified")
	}
	if b.p.attempts == 0 {
		panic("max number of retries has not been specified")
	}
	if b.p.onRetry == nil {
		panic("retry handler must be specified")
	}
	return &b.p
}
