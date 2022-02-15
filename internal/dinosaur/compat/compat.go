package compat

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/public"
)

// We expose some internal types here for compatability with code that still under `pkg`,
// TODO: figure out how to avoid exposing these here...

type Error = public.Error
type GenericOpenAPIError = public.GenericOpenAPIError
type PrivateError = private.Error
type WatchEvent = private.WatchEvent
type ErrorList = public.ErrorList
type ObjectReference = public.ObjectReference

var ContextAccessToken = public.ContextAccessToken
