// Package compat ...
package compat

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
)

// We expose some internal types here for compatability with code that still under `pkg`,
// TODO: figure out how to avoid exposing these here...

// Error ...
type Error = public.Error

// GenericOpenAPIError ...
type GenericOpenAPIError = public.GenericOpenAPIError

// PrivateError ...
type PrivateError = private.Error

// WatchEvent ...
type WatchEvent = private.WatchEvent

// ErrorList ...
type ErrorList = public.ErrorList

// ObjectReference ...
type ObjectReference = public.ObjectReference

// ContextAccessToken ...
var ContextAccessToken = public.ContextAccessToken
