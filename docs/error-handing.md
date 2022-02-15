# Error Handling in fleet-manager

## Overview

This document provides the best practices all developers should follow when handling errors in fleet-manager. 
At the time of writing this document (Apr 29, 2021), the code base doesn't follow these best practices. It is not required to change the current code base to follow the new practices. Instead, the practices should be adopted when new code are introduced, and if existing code needs to be updated for new changes, it should be refactored to follow the practice as part of the new work.

## Best Practices

### Use `github.com/pkg/errors` library to create errors

Use and only use functions like `errors.Errorf()` or `errors.Wrap()` from the [github.com/pkg/errors](https://github.com/pkg/errors) library to create new error values. This will ensure stack trace is added to the error properly.

#### Do

```
// example 1
err := errors.Errorf("this is an error")

// example 2
if err := somefunc(); err != nil {
    return errors.Wrap(err, "got an error from somefunc")
}
```

#### Don't

```
// don't use fmt.Errorf to create an error value
err := fmt.Errorf("this is an error")

// or this to wrap an error
err := fmt.Errorf("wrap error: %w", err)
```

### Capture the original error when creating a new ServiceError

When a new [ServiceError](../pkg/errors/errors.go) is created and it is caused by the another error, use the `NewWithCause` function to create the new one, but retain the original error value.
Make sure the message for the `Reason` field of the new `ServiceError` does not leak any internal information.

#### Do

```
// example

if err := somefunc(); err != nil {
    return NewWithCause(ErrorGeneral, err, "unexpected error from somefunc()")
}
```


#### Don't

```
// don't include the original error message in the new ServiceError
if err := somefunc(); err != nil {
    return NewWithCause(ErrorGeneral, err, "unexpected error: %v", err) // this will likely leak internal information to the Reason field
}
```

### Errors should be logged and forward to Sentry in central places

There is no need to log an error or forward it to Sentry at the place where it occurs. We have central places to handle errors.
The main places where errors handled are:

1. The [handError](../pkg/handlers/framework.go#L42) function. All errors for HTTP requested should be handled here, and it will log the error to logs and forward the error to Sentry.
2. The [runReconcile](../pkg/workers/reconciler.go#L87) function. All errors occur in the background workers should be handled here.
3. If the error is not returned to the caller, we should use an instance of the `UHCLogger` to log the error which will make sure it is captured by Sentry as well.

#### Do

```
// example
if err := somefunc(); err != nil {
    // just return the error to the caller
    return NewWithCause(ErrorGeneral, err, "unexpected error from somefunc()")
}

// example of error not returned to the caller
err := someFunc();
logger.Logger.Error(err)
```

#### Don't

```
// no need to log the error or capture it in Sentry where it occurs
if err := somefunc(); err != nil {
    golg.Errorf("unexpected error %v", err)
    sentry.CaptureExeception(err)
    return NewWithCause(ErrorGeneral, err, "unexpected error from somefunc()")
}

// example of error not returned to the caller
err := someFunc();
glog.Error(err)
```
