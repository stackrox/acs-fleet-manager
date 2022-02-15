/*
Copyright (c) 2018 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file contains functions that simplify generation of API metadata.

package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
)

// ServeHTTP sends API documentation response.
func (m *Metadata) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Set the content type:
	w.Header().Set("Content-Type", "application/json")

	// Prepare the body:
	body := Metadata{
		ID:   m.ID,
		Kind: "API",
		HREF: r.URL.Path,
	}
	for _, v := range m.Versions {
		href := v.HREF
		if href == "" {
			href = v.ID
		}
		body.Versions = append(body.Versions, VersionMetadata{
			ID:   v.ID,
			Kind: "APIVersion",
			HREF: fmt.Sprintf("%s/%s", r.URL.Path, href),
		})
	}

	data, err := json.Marshal(body)
	if err != nil {
		SendPanic(w, r)
		return
	}

	// Send the response:
	_, err = w.Write(data)
	if err != nil {
		err = fmt.Errorf("Can't send response body for request '%s'", r.URL.Path)
		glog.Error(err)
		sentry.CaptureException(err)
		return
	}
}

// ServeHTTP sends API version v1 documentation response.
func (v *VersionMetadata) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set the content type:
	w.Header().Set("Content-Type", "application/json")

	// Prepare the body:
	body := VersionMetadata{
		ID:   v.ID,
		Kind: "APIVersion",
		HREF: r.URL.Path,
	}
	for _, c := range v.Collections {
		href := c.HREF
		if href == "" {
			href = c.ID
		}
		body.Collections = append(body.Collections, CollectionMetadata{
			ID:   c.ID,
			Kind: c.Kind,
			HREF: fmt.Sprintf("%s/%s", r.URL.Path, href),
		})
	}

	data, err := json.Marshal(body)
	if err != nil {
		SendPanic(w, r)
		return
	}

	// Send the response:
	_, err = w.Write(data)
	if err != nil {
		glog.Errorf("Can't send response body for request '%s'", r.URL.Path)
		return
	}
}
