package k8s

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsCentralTLSNotFound(t *testing.T) {

	tests := map[string]struct {
		inputErr    error
		shouldMatch bool
	}{
		"match": {
			inputErr:    &SecretNotFound{SecretName: CentralTLSSecretName}, // pragma: allowlist secret
			shouldMatch: true,
		},
		"wrapped match": {
			inputErr:    fmt.Errorf("wrapped error: %w", &SecretNotFound{SecretName: CentralTLSSecretName}),
			shouldMatch: true,
		},
		"type does not match": {
			inputErr:    errors.New("some error"),
			shouldMatch: false,
		},
		"secret name does not match": {
			inputErr:    &SecretNotFound{SecretName: "test-name"}, // pragma: allowlist secret
			shouldMatch: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			actual := IsCentralTLSNotFound(tc.inputErr)
			assert.Equal(t, tc.shouldMatch, actual, "unexpected return value want: %s, got: %s", tc.shouldMatch, actual)
			if tc.shouldMatch != actual {
				t.Fail()
			}
		})
	}
}
