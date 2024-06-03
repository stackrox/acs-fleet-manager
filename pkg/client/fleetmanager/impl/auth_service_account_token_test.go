package impl

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServiceAccountTokenAuth(t *testing.T) {
	tokenDir := t.TempDir()
	tokenFile := writeTokenFile(t, tokenDir, "token")

	ticker := newTicker("2024-05-21T14:26:05Z")

	auth := serviceAccountTokenAuth{
		file:   tokenFile,
		leeway: 5 * time.Second,
		period: 6 * time.Minute,
		now:    ticker.tick,
	}

	token, err := auth.getOrLoadToken()
	require.NoError(t, err)

	require.Equal(t, "token", token)
}

func TestServiceAccountTokenAuth_Cache(t *testing.T) {
	tokenDir := t.TempDir()
	tokenFile := writeTokenFile(t, tokenDir, "token")

	ticker := newTicker(
		"2024-05-21T14:26:05Z",
		"2024-05-21T14:28:32Z",
	)

	auth := serviceAccountTokenAuth{
		file:   tokenFile,
		leeway: 5 * time.Second,
		period: 6 * time.Minute,
		now:    ticker.tick,
	}

	_, _ = auth.getOrLoadToken()
	_ = writeTokenFile(t, tokenDir, "don't read me")

	ticker.fastForward(t)

	token, err := auth.getOrLoadToken()
	require.NoError(t, err)

	require.Equal(t, "token", token)
}

func TestServiceAccountTokenAuth_Refresh(t *testing.T) {
	creationTime := "2024-05-21T00:00:00Z"

	tests := []struct {
		name     string
		readTime string
	}{
		{
			name:     "should reload between refresh and expiry",
			readTime: "2024-05-21T00:43:00Z",
		},
		{
			name:     "should reload between after expiry",
			readTime: "2024-05-21T00:43:00Z",
		},
		{
			name:     "should reload right before refresh (leeway)",
			readTime: "2024-05-21T00:05:57Z",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tokenDir := t.TempDir()
			tokenFile := writeTokenFile(t, tokenDir, "token1")

			ticker := newTicker(
				creationTime,
				test.readTime,
			)

			auth := serviceAccountTokenAuth{
				file:   tokenFile,
				leeway: 5 * time.Second,
				period: 6 * time.Minute,
				now:    ticker.tick,
			}

			token, err := auth.getOrLoadToken()
			require.NoError(t, err)
			require.Equal(t, "token1", token)

			_ = writeTokenFile(t, tokenDir, "token2")

			ticker.fastForward(t)

			token, err = auth.getOrLoadToken()
			require.NoError(t, err)
			require.Equal(t, "token2", token)
		})
	}
}

func writeTokenFile(t *testing.T, dir, contents string) string {
	t.Helper()
	filePath := filepath.Join(dir, "token")
	err := os.WriteFile(filePath, []byte(contents), 0o777)
	require.NoError(t, err)
	return filePath
}

type ticker struct {
	ticks []time.Time
	idx   int
}

func newTicker(timeSeries ...string) *ticker {
	ticks := make([]time.Time, len(timeSeries))
	for i, dateTime := range timeSeries {
		ticks[i], _ = time.Parse(time.RFC3339, dateTime)
	}
	return &ticker{
		ticks: ticks,
	}
}

func (tr *ticker) fastForward(t *testing.T) {
	require.Less(t, tr.idx, len(tr.ticks)-1)
	tr.idx++
}

func (tr *ticker) tick() time.Time {
	return tr.ticks[tr.idx]
}
