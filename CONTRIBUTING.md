# Contributing

## New contributors

 - Follow the [getting started guide](https://github.com/stackrox/acs-fleet-manager#getting-started) to setup your environment
 - Opening a PR follow the [PR template](https://github.com/stackrox/acs-fleet-manager/blob/main/.github/pull_request_template.md)
 - A pull-request must be approved by at least **one** engineer

## Specialities

### Dinosaurs

The `dinosaur` is a placeholder for the service name. It originated from the [fleet-manager template](https://github.com/bf2fc6cc711aee1a0c2a/ffm-fleet-manager-go-template).
Long-term all `dinosaur` occurrences will be replaced with our product name.

## Project Source
Project source is to be found under `$GOPATH/src` by a distinct directory path.
```plain
$GOPATH
  /bin
  /pkg
  /src
    /github.com/bf2fc6cc711aee1a0c2a/
      /fleet-manager -- our git root
        /cmd
          /fleet-manager  -- Main CLI entrypoint
        /internal   -- service specific implementations
           /dinosaur
               providers.go -- dinosaurs service injection setup
              /test  -- integration test folder
              /internal
                /services -- dinosaurs services
                /workers  -- dinosaurs workers
                /api      -- generated data transfer objects for the API and database entities
                /migrations -- dinosaurs database migrations
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
See the [adding-a-new-endpoint](./docs/adding-a-new-endpoint.md) documentation.

## Adding New Serve Command Flags
See the [Adding Flags to Fleet Manager](./docs/adding-new-flags.md) documentation for more information.

## Testing
See the [automated testing](./docs/automated-testing.md) documentation.

## Logging Standards & Best Practices
  * Log only actionable information, which will be read by a human or a machine for auditing or debugging purposes
    * Logs shall have context and meaning - a single log statement should be useful on its own
    * Logs shall be easily aggregatable
    * Logs shall never contain sensitive information
  * All logs should be logged through our logging interface, `UHCLogger` in `/pkg/logger/logger.go`
    * *Logging interface shall be updated to gracefully handle logs outside of a user context*
  * If a similar log message will be used in more than one place, consider adding a new standardized interface to `UHCLogger`
    * *Logging interface shall be updated to define a new `Log` struct to support standardization of more domain specific log messages*

### Levels
#### Info
Log to this level any non-error based information that might be useful to someone browsing logs for a specific reason. This may or may not include request / response logging, debug information, script output, etc.

#### Warn
Log to this level any error based information that might want to be brought to someone's attention to take action on, but does not seriously impede or affect use of the application (ie. it is recoverable). This may or may not include deprecation notices, retry operations, etc.

#### Error
Log to this level any error that is fatal to the given transaction and affects expected user operation. This may or may not include failed connections, missing expected data, or other unrecoverable outcomes.
Error handling should be implemented by following these best practices as laid out in [this guide](docs/error-handing.md).

#### Fatal
Log to this level any error that is fatal to the service and requires the service to be immediately shutdown in order to prevent data loss or other unrecoverable states. This should be limited to scripts and fail-fast scenarios in service startup *only* and *never* because of a user operation in an otherwise healthy servce.

### Verbosity
Verbosity effects the way in which `Info` logs are written. The best way to see how verbosity works is here: https://play.golang.org/p/iXJiX289VzO

On a scale from 1 -> 10, logging items at `V(10)` would be considered something akin to `TRACE` level logging, whereas `V(1)` would be information you might want to log all of the time.

As a rule of thumb, we use verbosity settings in the following ways. Consider we have:

```go
glog.V(1).Info("foo")
glog.V(5).Info("bar")
glog.V(10).Info("biz")
```
* `--v=1`
  * This is production level logging. No unecessary spam and no sensitive information.
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

## Running linter

`golangci-lint` is used to run a static code analysis. This is enabled on per MR check and it will fail, if any new changes don't conform to the rules specified by the linter.

To manually run the check, execute this command from the root of this repository

```sh
make lint
```
## Writing Docs

Please see the [README](./docs/README.md) in `docs` directory.
