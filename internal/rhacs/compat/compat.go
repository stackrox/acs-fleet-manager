package compat

import (
	"github.com/stackrox/acs-fleet-manager/internal/rhacs/internal/api/public"
)

// We expose some internal types here for compatability with code that still under `pkg`,
// TODO: figure out how to avoid exposing these here...

type Error = public.Error
type GenericOpenAPIError = public.GenericOpenAPIError
type ErrorList = public.ErrorList
type ObjectReference = public.ObjectReference

var ContextAccessToken = public.ContextAccessToken
