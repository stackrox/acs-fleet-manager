//go:build tools
// +build tools

package tools

// This file declares dependencies on tool for `go mod` purposes.
// See https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
// for an explanation of the approach.

import (
	// Tool dependencies, not used anywhere in the code.
	_ "github.com/matryer/moq"
	_ "github.com/segmentio/chamber/v2"
	_ "gotest.tools/gotestsum"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
