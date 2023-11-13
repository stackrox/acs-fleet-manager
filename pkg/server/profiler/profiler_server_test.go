package profiler

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPprofProfiler(t *testing.T) {
	server := SingletonPprofServer()
	server.Start()

	for {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", "6060"), 5*time.Second)
		require.NoError(t, err)
		if conn != nil {
			require.NoError(t, conn.Close())
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Test server was stopped
	server.Stop()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", "6060"), 5*time.Second)
	require.Error(t, err)
	require.Nil(t, conn)
}
