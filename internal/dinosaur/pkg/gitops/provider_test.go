package gitops

import (
	"sync/atomic"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Get(t *testing.T) {

	var failingValidation validationFn = func(config Config) error {
		return assert.AnError
	}
	var successfulValidation validationFn = func(config Config) error {
		return nil
	}
	var failingReader Reader = &mockReader{err: assert.AnError}
	var successfulReader Reader = &mockReader{config: Config{}}

	type tc struct {
		name                     string
		hasLastWorkingConfig     bool
		reader                   Reader
		validator                validationFn
		expectedErrorMetricCount int
		expectError              bool
	}
	tcs := []tc{
		{
			name:                     "Successful without last working config",
			hasLastWorkingConfig:     false,
			reader:                   successfulReader,
			validator:                successfulValidation,
			expectedErrorMetricCount: 0,
			expectError:              false,
		}, {
			name:                     "Successful with last working config",
			hasLastWorkingConfig:     true,
			reader:                   successfulReader,
			validator:                successfulValidation,
			expectedErrorMetricCount: 0,
			expectError:              false,
		}, {
			name:                     "reader fails without last working config",
			hasLastWorkingConfig:     false,
			reader:                   failingReader,
			validator:                successfulValidation,
			expectedErrorMetricCount: 1,
			expectError:              true,
		}, {
			name:                     "reader fails with last working config",
			hasLastWorkingConfig:     true,
			reader:                   failingReader,
			validator:                successfulValidation,
			expectedErrorMetricCount: 1,
			expectError:              false,
		}, {
			name:                     "Validation fails without last working config",
			hasLastWorkingConfig:     false,
			reader:                   failingReader,
			validator:                failingValidation,
			expectedErrorMetricCount: 1,
			expectError:              true,
		}, {
			name:                     "Validation fails with last working config",
			hasLastWorkingConfig:     true,
			reader:                   failingReader,
			validator:                failingValidation,
			expectedErrorMetricCount: 1,
			expectError:              false,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			p := &provider{}
			p.lastWorkingConfig = atomic.Pointer[Config]{}

			if tc.hasLastWorkingConfig {
				// Get the config once to set the last working config
				p.reader = successfulReader
				p.validationFn = successfulValidation
				_, err := p.Get()
				require.NoError(t, err)
			}

			p.reader = tc.reader
			p.validationFn = tc.validator

			metrics.GitopsConfigProviderErrorCounter.Reset()
			_, err := p.Get()
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			count := testutil.CollectAndCount(metrics.GitopsConfigProviderErrorCounter)
			assert.Equal(t, tc.expectedErrorMetricCount, count)

		})
	}
}

type mockReader struct {
	config Config
	err    error
}

func (r *mockReader) Read() (Config, error) {
	return r.config, r.err
}

func (r *mockReader) WillFail() *mockReader {
	r.err = assert.AnError
	return r
}

func (r *mockReader) WillSucceed() *mockReader {
	r.err = nil
	return r
}

var _ Reader = &mockReader{}
