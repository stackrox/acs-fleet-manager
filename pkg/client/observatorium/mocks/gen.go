package mocks

import pV1 "github.com/prometheus/client_golang/api/prometheus/v1"

// API an alias for pV1.API
//
//go:generate moq -rm -out api_moq.go . API
type API = pV1.API
