# Contributing

## New contributors

 - Follow the [getting started guide](https://github.com/stackrox/acs-fleet-manager#getting-started) to setup your environment
 - Pull-Requests:
   - check the [PR template](https://github.com/stackrox/acs-fleet-manager/blob/main/.github/pull_request_template.md) checkboxes
   - must be approved by at least **one** engineer
   - title contains ticket number: `ROX-12345: My PR title`

## Specialities

### Go package structure

Project source is to be found under `$GOPATH/src` by a distinct directory path.
```plain
/fleet-manager -- our git root
/cmd
  /fleet-manager  -- Main CLI entrypoint
/internal   -- service specific implementations
   /central -- represents ACS Central component (central, external central-db, scanner, scanner-db)
       providers.go -- centrals service injection setup
      /test  -- integration test folder
      /internal
        /services -- central services
        /workers  -- central workers
        /api      -- generated data transfer objects for the API and database entities
        /migrations -- database migrations
        /presenters -- DTO converters and presenters
        /routes  -- routes setup
        /environments -- environment setup
        /handlers -- api endpoint handlers
/pkg
  /api      -- type definitions and models (Note. openapi folder is generated - see below)
  /config   -- configuration handling
  /db  		 -- database schema and migrations
  /handlers -- web handlers/controllers
  /services -- interfaces for CRUD and business logic
    /syncsetresources -- resource definitions to be created via syncset
  /workers  -- background workers for async reconciliation logic

```

## Debugging

### VS Code
Set the following configuration in your **Launch.json** file.
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Fleet Manager API",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/fleet-manager/main.go",
            "env": {
                "OCM_ENV": "development"
            },
            "args": ["serve"]
        }
    ]
}
```

## Modifying the API definition

The services' OpenAPI specification is located in `openapi/fleet-manager.yaml`. It can be modified using Apicurio Studio, Swagger or manually.

Once you've made your changes, the second step is to validate it:

```sh
make openapi/validate
```

Once the schema is valid, the remaining step is to generate the openapi modules by using the command:

```sh
make openapi/generate
```

## Adding a new endpoint
See the [adding-a-new-endpoint](./docs/development/adding-a-new-endpoint.md) documentation.

## Logging Standards & Best Practices
  * Log only actionable information, which will be read by a human or a machine for auditing or debugging purposes
    * Logs shall have context and meaning - a single log statement should be useful on its own
    * Logs shall be easily aggregatable
    * Logs shall never contain sensitive information
  * All logs should be logged through our logging interface, `UHCLogger` in `/pkg/logger/logger.go`
    * *Logging interface shall be updated to gracefully handle logs outside of a user context*
  * If a similar log message will be used in more than one place, consider adding a new standardized interface to `UHCLogger`
    * *Logging interface shall be updated to define a new `Log` struct to support standardization of more domain specific log messages*

### Verbosity
On a scale from 1 -> 10, logging items at `V(10)` would be considered something akin to `TRACE` level logging,
whereas `V(1)` would be information you might want to log all of the time.

We use verbosity settings in the following ways:
```go
glog.V(1).Info("foo")
glog.V(5).Info("bar")
glog.V(10).Info("biz")
```
* `--v=1`
  * This is production level logging. No unnecessary spam and no sensitive information.
  * This means that given the verbosity setting and the above code, we would see `foo` logged.
* `--v=5`
  * This is stage / test level logging. Useful debugging information, but not spammy. No sensitive information.
  * This means that given the verbosity setting and the above code, we would see `foo` and `bar` logged.
* `--v=10`
  * This is local / debug level logging. Useful information for tracing through transactions on a local machine during development.
  * This means that given the verbosity setting and the above code, we would see `foo`, `bar`, and `biz` logged.

### Sentry Logging
Sentry monitors errors/exceptions in a real-time environment. It provides detailed information about captured errors. See [sentry](https://sentry.io/welcome/) for more details.

Logging can be enabled by importing the sentry-go package: "github.com/getsentry/sentry-go

Following are possible ways of logging events via Sentry:

```go
sentry.CaptureMessage(message) // for logging message
sentry.CaptureEvent(event) // capture the events
sentry.CaptureException(error) // capture the exception
```
Example :
```go
func check(err error, msg string) {
	if err != nil && err != http.ErrServerClosed {
		glog.Errorf("%s: %s", msg, err)
		sentry.CaptureException(err)
	}
}
```

## Writing Docs

Please see the [docs](./docs) directory.
