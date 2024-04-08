package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfig_Success(t *testing.T) {
	t.Setenv("EMAIL_SENDER_SERVER_ADDRESS", ":8888")
	t.Setenv("EMAIL_SENDER_METRICS_ADDRESS", ":9999")

	cfg, err := GetConfig()

	require.NoError(t, err)
	assert.Equal(t, cfg.StartupTimeout, 300*time.Second)
	assert.Equal(t, cfg.ServerAddress, ":8888")
	assert.Equal(t, cfg.MetricsAddress, ":9999")
}
